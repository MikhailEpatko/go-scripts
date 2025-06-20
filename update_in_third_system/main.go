package main

import (
	"bytes"
	cm "common"
	iterjson "ezpkg.io/iter.json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	log.Println("Started")
	files, err := filepath.Glob(cm.WorkDir + "*.json")
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		parseAndSend(file)
	}
	log.Println("Finished")
}

func parseAndSend(file string) {
	var data = parseData(file)
	var chunk = make(map[string]string)
	var i = 0
	for k, v := range data.Pairs {
		if i >= cm.ChunkSize {
			send(data.Keyset, &chunk)
			i = 0
			chunk = make(map[string]string)
		}
		chunk[k] = v
		i++
	}
	send(data.Keyset, &chunk)
}

// parseData load Data from JSON file
func parseData(file string) *cm.Data {
	var jsonFile, err = os.Open(file)
	if err != nil {
		log.Fatalf("error opening file: %s", file)
	}
	defer func() {
		var err = jsonFile.Close()
		if err != nil {
			log.Fatalf("error closing file: %s", file)
		}
	}()
	ba, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("error reading jsonFile: %s", err.Error())
	}
	var data cm.Data
	err = cm.Json.Unmarshal(ba, &data)
	if err != nil {
		log.Fatalf("error unmarshaling json file: %s\n", file)
	}
	data.Keyset = cm.KeysetPrefix + data.Keyset
	return &data
}

// send make POST request to the third system - send chunk of key-value Pairs.
// JSON body example:
//
//	{
//	   <keyset>: {
//	     <key>: <value>
//	   },
//	}  
func send(
	keyset string,
	chunk *map[string]string,
) {
	b := iterjson.NewBuilder("", "  ")
	b.Add("", iterjson.TokenObjectOpen)
	b.Add(keyset, iterjson.TokenObjectOpen)
	for key, value := range *chunk {
		b.Add(key, value)
	}
	b.Add("", iterjson.TokenObjectClose)
	b.Add("", iterjson.TokenObjectClose)
	var ba, err = b.Bytes()
	var bb = bytes.NewBuffer(ba)
	request, err := http.NewRequest(http.MethodPost, cm.ThirdSystemUrl, bb)
	if err != nil {
		log.Fatalf("error creating request: %s", err.Error())
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("accept", "application/json")
	request.Header.Add("Authorization", cm.AuthToken)
	resp, err := cm.HttpClient.Do(request)
	if err != nil {
		log.Fatalf("error sending request: %v %s", *request, err.Error())
	}
	defer func() {
		var err = resp.Body.Close()
		if err != nil {
			log.Fatalf("error closing response body: %v, %s", resp, err.Error())
		}
	}()
	log.Println("response status code:", resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading response body:", err.Error())
	} else {
		log.Println(string(body))
	}
}
