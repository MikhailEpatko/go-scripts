package common

import "encoding/json"

type Table struct {
	Name    string
	Cache   string
	GetUrl  string
	SaveUrl string
}

type JsonId struct {
	Id int `json:"id"`
}

type JsonSource struct {
	Sources []json.RawMessage `json:"items"`
}

type Data struct {
	Keyset string            `json:"Keyset"`
	Pairs  map[string]string `json:"Pairs"`
}
