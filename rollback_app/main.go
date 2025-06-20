package main

import (
	cm "common"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

func main() {
	log.Println("rolling back started", cm.WorkDir)
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
		go deleteNewJsons(wg, table)
	}
	wg.Wait()
	log.Printf("rolling back finished")
}

// deleteNewJsons - удалить через API новые JSON-ы
func deleteNewJsons(
	wg *sync.WaitGroup,
	table cm.Table,
) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%s: deleteNewJsons: recovered: %v\n", table.Name, r)
		}
		wg.Done()
	}()
	var ids, err = getIdsFromFile(table)
	if err != nil {
		log.Printf("%s: deleteNewJsons: %v\n", table.Name, err)
		return
	}
	for _, id := range ids {
		if id != "" {
			err = deleteJson(table, id)
			if err != nil {
				log.Printf("%s: id = %s: deleteNewJsons: %v\n", table.Name, id, err)
			}
		}
	}
	log.Printf("%s: deleteNewJsons: finished\n", table.Name)
}

// getIdsFromFile - загрузить id JSON-ов из файлов
func getIdsFromFile(table cm.Table) ([]string, error) {
	var filename = cm.WorkDir + table.Name + "-new-ids.txt"
	var ba, err = os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%s: getIdsFromFile: reading file error: %v", table.Name, err)
	}
	return strings.Split(string(ba), " "), nil
}

// deleteJson - удадляет новый JSON
func deleteJson(
	table cm.Table,
	id string,
) error {
	request, err := http.NewRequest(http.MethodDelete, table.SaveUrl+"/"+id, nil)
	if err != nil {
		return fmt.Errorf("deleteJson: id = %s: request creation error: %w", id, err)
	}
	request.Header.Add(cm.TvmHeader, cm.TvmValue)
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("accept", "application/json")
	resp, err := cm.HttpClient.Do(request)
	if err != nil {
		return fmt.Errorf("deleteJson: id = %s: request sending error: %w", id, err)
	}
	defer func() {
		var err = resp.Body.Close()
		if err != nil {
			log.Printf("%s: deleteJson: id = %s: closing response body error: %v", table.Name, id, err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("deleteJson: id = %s: reading response body error: %w", id, err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf(
			"deleteJson: id = %s: unexpected response status code: %d response body: %s\n",
			id,
			resp.StatusCode,
			string(body),
		)
	} else {
		println("deleted:", id, "response:", string(body))
	}
	return nil
}
