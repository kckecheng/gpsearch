package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/urfave/cli/v2"
)

// Const definition
const (
	goDocEP = "https://api.godoc.org/search?q=" // // godoc.org API endpoint
	ErrExit = 1                                 // ErrExit code for abnormal exist
)

/* Query Result shares below format:
{
    "results": [
        {
            "name": "client",
            "path": "github.com/micro/go-micro/client",
            "import_count": 971,
            "synopsis": "Package client is an interface for an RPC client",
            "stars": 10770,
            "score": 0.99
        },
        {
            "name": "registry",
            "path": "github.com/micro/go-micro/registry",
            "import_count": 717,
            "synopsis": "Package mdns is a multicast dns registry",
            "stars": 10750,
            "score": 0.99
				"fork": true
		}
	]
} */
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

func formatOut(pkgs []map[string]interface{}, fields []string) {
	for _, pkg := range pkgs {
		for _, field := range fields {
			v, ok := pkg[field]
			if !ok {
				if field == "fork" {
					fmt.Println("fork: false")
				} else if field == "stars" {
					fmt.Println("stars: 0")
				} else {
					fmt.Printf("%s: n/a\n", field)
				}
			} else {
				fmt.Printf("%s: %v\n", field, v)
			}
		}
		if len(fields) > 1 {
			fmt.Println()
		}
	}
}

func sortPkgs(pkgs []map[string]interface{}, field string) {
	greater := func(i, j int) bool {
		v1, ok1 := pkgs[i][field]
		v2, ok2 := pkgs[j][field]
		if !ok1 && !ok2 {
			return true // Do not change the order if both records do not exist
		} else if ok1 && !ok2 {
			return true
		} else if !ok1 && ok2 {
			return false
		}

		switch v1.(type) {
		case string:
			// Reverse order for string: a goes first then b, etc.
			return v1.(string) < v2.(string)
		case float32:
			return v1.(float32) > v2.(float32)
		case float64:
			return v1.(float64) > v2.(float64)
		case int32:
			return v1.(int32) > v2.(int32)
		case int64:
			return v1.(int64) > v2.(int64)
		case bool:
			if v1.(bool) || (!v1.(bool) && !v2.(bool)) {
				return true
			}
			return false
		}

		return false
	}
	sort.Slice(pkgs, greater)
}

func main() {
	app := &cli.App{
		Name:        "gpsearch",
		Usage:       "search golang packages",
		Description: "search golang packages on godoc.org from CLI",
		Version:     "1.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "sort",
				Aliases: []string{"s"},
				Value:   "import_count",
				Usage:   "sort packages based on",
			},
			&cli.BoolFlag{
				Name:    "reverse",
				Aliases: []string{"r"},
				Value:   false,
				Usage:   "reverse the sort result",
			},
			&cli.Uint64Flag{
				Name:    "num",
				Aliases: []string{"n"},
				Value:   10,
				Usage:   "num. of packages to list",
			},
			&cli.StringSliceFlag{
				Name:    "fields",
				Aliases: []string{"f"},
				Value:   cli.NewStringSlice("path", "import_count", "synopsis"),
				Usage:   "package fields to show, speficie multiple values with -f <field1> -f <field2> ...",
			},
		},
		Action: func(c *cli.Context) error {
			// Exit if no query string is provided
			if c.NArg() == 0 {
				fmt.Printf("\nNo query string is provided\n\n")
				cli.ShowAppHelpAndExit(c, ErrExit)
			}

			// Query for packages
			qstr := strings.Join(c.Args().Slice(), "20%")
			pkgs, err := query(qstr)
			if err != nil {
				return err
			}

			// Sort packages
			sortPkgs(pkgs, c.String("sort"))
			// Reverse if needed
			if c.Bool("reverse") {
				rpkgs := []map[string]interface{}{}
				for i := len(pkgs) - 1; i >= 0; i-- {
					rpkgs = append(rpkgs, pkgs[i])
				}
				pkgs = rpkgs
			}

			// Slice packages based on specified num.
			var results []map[string]interface{}
			ns := c.Uint64("num")
			np := uint64(len(pkgs))
			if ns < np {
				results = pkgs[:ns]
			} else {
				results = pkgs
			}

			// Output
			formatOut(results, c.StringSlice("fields"))
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
