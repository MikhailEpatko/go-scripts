package main

import (
	cm "common"
	iterjson "ezpkg.io/iter.json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

func main() {
	log.Println("started")
	var wg = &sync.WaitGroup{}
	for _, table := range cm.Tables {
		wg.Add(1)
		go createTranslationFiles(wg, table)
	}
	wg.Wait()
	log.Printf("finished")
}

// createTranslationFiles - создать файлы с переводами
func createTranslationFiles(
	wg *sync.WaitGroup,
	table cm.Table,
) {
	defer wg.Done()
	jsonFiles, err := loadJsonFiles(table)
	if err != nil {
		log.Printf("%s: createTranslationFiles: %v\n", table.Name, err)
		return
	}
	keyset, err := createKeyset(table.Name, jsonFiles)
	if err != nil {
		log.Printf("%s: createTranslationFiles: %v\n", table.Name, err)
		return
	}
	err = writeFile(table.Name, keyset)
	if err != nil {
		log.Printf("%s: createTranslationFiles: %v\n", table.Name, err)
	}
	log.Println(table.Name, "createTranslationFiles: file creating finished")
}

// loadJsonFiles - загрузить JSON-файлы из приложения
func loadJsonFiles(table cm.Table) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, table.GetUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: loadJsonFiles: creating request error: %w", table.Name, err)
	}
	request.Header.Add(cm.TvmHeader, cm.TvmValue)
	resp, err := cm.HttpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%s: loadJsonFiles: sending request error: %v %w", table.Name, *request, err)
	}
	defer func() {
		var err = resp.Body.Close()
		if err != nil {
			log.Printf("%s: loadJsonFiles: closing response body error: %v, %v", table.Name, resp, err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: loadJsonFiles: reading response body error: %v\n", table.Name, err)
	}
	log.Println(table.Name, "loadJsonFiles: response status code:", resp.StatusCode)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s: loadJsonFiles: response status code error: %v\n", table.Name, string(body))
	}
	return body, nil
}

// createKeyset - создать набор пар ключ - перевод
func createKeyset(
	keyset string,
	source []byte,
) ([]byte, error) {
	var i, id int
	b := iterjson.NewBuilder("", "  ")
	b.Add("", iterjson.TokenObjectOpen)
	b.Add("name", keyset)
	b.Add("pairs", iterjson.TokenObjectOpen)
	for item, err := range iterjson.Parse(source) {
		if err != nil {
			return nil, fmt.Errorf("%s: createKeyset: parsing source error: %w", keyset, err)
		}
		err = func() error {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Recovered: ", r)
				}
			}()
			var path, key, token = item.GetPathString(), item.Key, item.Token
			path = cm.JsonItemsPrefix.ReplaceAllString(path, "")
			if path == "id" {
				id, err = token.GetInt()
				if err != nil {
					return fmt.Errorf("%s: createKeyset: parsing id error: %w", keyset, err)
				}
			}
			if !(skipByPath(path) || key.IsZero() || token.Type() != iterjson.TokenString || skipByValue(token.String())) {
				i++
				var k = fmt.Sprintf("%d.%s", id, path)
				b.Add(k, token)
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}
	b.Add("", iterjson.TokenObjectClose)
	b.Add("", iterjson.TokenObjectClose)
	out, err := b.Bytes()
	if err != nil {
		return nil, fmt.Errorf("%s: createKeyset: building json bytes error: %w", keyset, err)
	}
	return out, nil
}

// skipByPath - проверка: нужно ли пропустить создание перевода для этого пути
func skipByPath(path string) bool {
	path = strings.ToLower(path)
	return strings.HasSuffix(path, "type") ||
		strings.HasSuffix(path, "coloroption") ||
		strings.HasSuffix(path, "color") ||
		!strings.Contains(path, "text") &&
			!strings.Contains(path, "title") &&
			!strings.Contains(path, "description") &&
			!strings.Contains(path, "caption")
}

// skipByValue - проверка: нужно ли пропустить создание перевода для этого значения
func skipByValue(value string) bool {
	return strings.HasPrefix(value, cm.ColorPattern) ||
		strings.Contains(value, cm.UrlPattern) ||
		strings.TrimSpace(value) == ""
}

// writeFile - записать кейсет в файл
func writeFile(
	keysetName string,
	keysetData []byte,
) error {
	f, err := os.Create(cm.WorkDir + keysetName + ".json")
	if err != nil {
		return fmt.Errorf("%s: writeFile: creating file error: %w", keysetName, err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.Printf("%s: writeFile: closing file error: %v", keysetName, err)
		}
	}()
	_, err = f.Write(keysetData)
	if err != nil {
		return fmt.Errorf("%s: writeFile: writing file error: %v", keysetName, err)
	}
	return nil
}
