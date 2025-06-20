package common

import (
	jsoniterator "github.com/json-iterator/go"
	"log"
	"net/http"
	"os"
	"regexp"
)

var (
	Tables = []Table{
		{
			Name:    "table1",
			Cache:   "table1-cache",
			GetUrl:  UrlPrefix + "/table1",
			SaveUrl: UrlPrefix + "/table1",
		},
		{
			Name:    "table2",
			Cache:   "table2-cache",
			GetUrl:  UrlPrefix + "/table2",
			SaveUrl: UrlPrefix + "/table2",
		},
	}
	UrlPrefix = "http://localhost:8082"
	// префикс для названий наборов пар ключ-значение
	KeysetPrefix = "testing-"
	// паттерн для удаления items.XX из пути JSON-а
	JsonItemsPrefix = regexp.MustCompile(`^items.\d+.`)
	ThirdSystemUrl  = "https://..."
	AuthToken       = "TOKEN"
	// количество пар ключ-значение в одном запросе, отправляемом в Танкер
	ChunkSize = 100
	// быстрый анмаршалинг JSON-ов
	Json       = jsoniterator.ConfigCompatibleWithStandardLibrary
	HttpClient = &http.Client{}
	// директория, в которой будут создаваться все файлы
	WorkDir = getWorkDir()
)

func getWorkDir() string {
	var wd, err = os.Getwd()
	if err != nil {
		log.Fatal("confg: getting working directory error: %v", err)
	}
	return wd + "/../files/"
}
