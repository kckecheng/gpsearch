package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const goDocEP = "https://api.godoc.org/search?q="

// Error code for abnormal exist
const (
	ErrGet   = 1
	ErrParse = 2
)

type queryResp struct {
	Results []struct {
		Name     string  `json:"name"`
		Path     string  `json:"path"`
		Import   int32   `json:"import_count"`
		Synopsis string  `json:"synopsis"`
		Starts   int32   `json:"stars"`
		Score    float32 `json:"score"`
	} `json:"results"`
}

func parseArgs() {
	var (
		n int    // Num. of records to show
		q string // query string
		f string // Format: table, list
		d bool   // Show description
		u bool   // Show num. of projects use this package
		h bool   // Show starts
		s bool   // Show score
	)
}

func main() {
	uri := "micro/go-micro"
	url := goDocEP + fmt.Sprintf("\"%s\"", uri)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failt to query %s\n", uri)
		os.Exit(ErrGet)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	data := queryResp{}
	err = decoder.Decode(&data)
	if err != nil {
		log.Printf("Fail to parse query results")
		os.Exit(ErrParse)
	}
	for _, record := range data.Results {
		fmt.Printf("Name: %s\n", record.Path)
		fmt.Printf("Description: %s\n", record.Synopsis)
		fmt.Printf("Used by: %d\n", record.Import)
		fmt.Printf("Starts: %d\n", record.Starts)
		fmt.Printf("Score: %.2f\n", record.Score)
		fmt.Println()
	}
}
