package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// Const definition
const (
	goDocEP = "https://api.godoc.org/search?q=" // // godoc.org API endpoint
	ErrExit = 1                                 // ErrExit code for abnormal exist
)

// Query Result shares below format:
// {
//     "results": [
//         {
//             "path": "github.com/micro/go-micro/client",
//             "import_count": 0,
//             "synopsis": "Package client is an interface for an RPC client"
//         }
//     ]
// }
func query(s string) ([]map[string]interface{}, error) {
	url := goDocEP + fmt.Sprintf("%s", s)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	data := map[string][]map[string]interface{}{}
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}
	return data["results"], nil
}

func main() {
	pkgs, err := query("micro")
	if err != nil {
		fmt.Println(err)
		os.Exit(ErrExit)
	}
	fmt.Printf("%+v", pkgs)
}
