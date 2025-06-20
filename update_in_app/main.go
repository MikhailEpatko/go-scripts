package main

import (
	"bytes"
	cm "common"
	"encoding/json"
	iterjson "ezpkg.io/iter.json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

func main() {
	log.Println("started")
	defer func() {
		var err = cm.SetCacheUpdateEnabled(true)
		if err != nil {
			log.Printf("main: error enable caches update: %v\n", err)
			return
		}
		log.Println("main: caches update enabled")
	}()
	err := cm.SetCacheUpdateEnabled(false)
	if err != nil {
		log.Printf("main: error disable caches update: %v\n", err)
		return
	}
	log.Println("main: caches update disabled")
	var wg = &sync.WaitGroup{}
	for _, table := range cm.Tables {
		wg.Add(1)
		go createNewJsons(wg, table)
	}
	wg.Wait()
	log.Printf("finished")
}

// createNewJsons - создать и сохранить в приложении новые JSON-ы
func createNewJsons(
	wg *sync.WaitGroup,
	table cm.Table,
) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%s: createNewJsons: recovered: %v\n", table.Name, r)
		}
		wg.Done()
	}()
	jssonFiles, err := downloadJsons(table)
	if err != nil {
		log.Printf("%s: createTranslationFiles: %v\n", table.Name, err)
		return
	}
	var data cm.Json
	err = json.Unmarshal(jssonFiles, &data)
	if err != nil {
		log.Printf("%s: createNewJsons: json unmarshaling error: %v\n", table.Name, err)
		return
	}
	var ids = make(chan int, 1)
	var idCollectorWg = &sync.WaitGroup{}
	idCollectorWg.Add(1)
	go collectIds(idCollectorWg, table, ids)
	for _, source := range data.Sources {
		newJson, id, err := createNewOne(table, source)
		if err != nil {
			log.Printf("%s: id = %d: createNewJsons: %v\n", table.Name, id, err)
			continue
		}
		err = uploadJson(table, newJson, ids)
		if err != nil {
			log.Printf("%s: id = %d: createNewJsons: %v\n", table.Name, id, err)
		}
	}
	close(ids)
	idCollectorWg.Wait()
	log.Printf("%s: createNewJsons: finished\n", table.Name)
}

// downloadJsons - загрузить JSON-ы из приложения
func downloadJsons(table cm.Table) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, table.GetUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("downloadJsons: creating request error: %w", err)
	}
	request.Header.Add(cm.TvmHeader, cm.TvmValue)
	resp, err := cm.HttpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("downloadJsons: sending request error: %v %w", *request, err)
	}
	defer func() {
		var err = resp.Body.Close()
		if err != nil {
			log.Printf("%s: downloadJsons: closing response body error: %v", table.Name, err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("downloadJsons: reading response body error: %v\n", err)
	}
	log.Println(table.Name, "downloadJsons: response status code:", resp.StatusCode)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("downloadJsons: forbidden response status code=%d %s\n", resp.StatusCode, string(body))
	}
	return body, nil
}

// createNewOne - создаёт новый JSON, в котором тексты заменены на ключи переводов
func createNewOne(
	table cm.Table,
	source []byte,
) ([]byte, int, error) {
	var id int
	var b = iterjson.NewBuilder("", "  ")
	for item, err := range iterjson.Parse(source) {
		if err != nil {
			return nil, 0, fmt.Errorf("createNewOne: source parsing error: %w", err)
		}
		var path, key, token = item.GetPathString(), item.Key, item.Token
		if path == "id" {
			id, err = token.GetInt()
			if err != nil {
				return nil, 0, fmt.Errorf("createNewOne: id parsing error: %w", err)
			}
			continue
		}
		if skipByPath(path) || key.IsZero() || token.Type() != iterjson.TokenString || skipByValue(token.String()) {
			b.Add(key, token)
		} else {
			var thirdSystemKey = fmt.Sprintf("%s%s.%d.%s", cm.KeysetPrefix, table.Name, id, path)
			b.Add(key, thirdSystemKey)
		}
	}
	out, err := b.Bytes()
	if err != nil {
		return nil, id, fmt.Errorf("createNewOne: getting bytes from json builder error: %w", err)
	}
	return out, id, nil
}

// skipByPath - проверяет: нужно ли пропустить создание перевода для этого пути
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

// skipByValue - проверяет: нужно ли пропустить создание перевода для этого значения
func skipByValue(value string) bool {
	return strings.HasPrefix(value, cm.ColorPattern) ||
		strings.Contains(value, cm.UrlPattern) ||
		strings.TrimSpace(value) == ""
}

// uploadJson - сохранение через API приложения нового JSON-а
func uploadJson(
	table cm.Table,
	out []byte,
	idChan chan<- int,
) error {
	request, err := http.NewRequest(http.MethodPost, table.SaveUrl, bytes.NewBuffer(out))
	if err != nil {
		return fmt.Errorf("uploadJson: request creating error: %w", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("accept", "application/json")
	resp, err := cm.HttpClient.Do(request)
	if err != nil {
		return fmt.Errorf("uploadJson: request sending error: %w", err)
	}
	defer func() {
		var err = resp.Body.Close()
		if err != nil {
			log.Printf("%s: uploadJson: closing response body error: %v", table.Name, err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("uploadJson: reading response body error: %w", err)
	}
	if resp.StatusCode == 200 {
		var cfg cm.JsonId
		err = json.Unmarshal(body, &cfg)
		if err != nil {
			log.Fatalf("uploadJson: unmarshaling JsonId error: %w", err)
		}
		idChan <- cfg.Id
	} else {
		return fmt.Errorf(
			"uploadJson: unexpected response status code: %d response body: %s\n",
			resp.StatusCode,
			string(body),
		)
	}
	return nil
}

// collectIds - собирает id новых JSON-ов в файлы <table.name>-new-ids.txt на случай, если нужно будет откатить изменения
func collectIds(
	wg *sync.WaitGroup,
	table cm.Table,
	ids <-chan int,
) {
	defer wg.Done()
	var b = strings.Builder{}
	for id := range ids {
		var _, err = b.WriteString(strconv.Itoa(id) + " ")
		if err != nil {
			log.Printf("%s: collectIds: adding id to the builder error: %v", table.Name, err)
		}
	}
	var filename = cm.WorkDir + table.Name + "-new-ids.txt"
	var err = os.WriteFile(filename, []byte(b.String()), 0644)
	if err != nil {
		log.Printf("%s: collectIds: writing file error: %v", table.Name, err)
	}
	log.Printf("%s: collectIds: new ids saved to the file '%s'", table.Name, filename)
}
