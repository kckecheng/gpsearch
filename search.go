package main

import (
	"crypto/sha1"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// Const definition
const (
	goDocEP = "https://api.godoc.org/search?q=" // godoc.org API endpoint
	ErrExit = 1                                 // ErrExit code for abnormal exist
)

/* Cache control varaibles:
cacheDir: where cache is stored (/tmp/gpsearch on Linux, %TEMP%\gpsearch on Windows)
cacheTimeOut: how long cached data will expire
*/
var cacheDir string
var cacheTimeOut int

// init cacheDir based on env GPSEARCH_CACHEDIR
func init() {
	d, ok := os.LookupEnv("GPSEARCH_CACHEDIR")
	if ok {
		if !exist(d) {
			log.Fatalf("The specified cache directory %s does not exist", d)
		}
		cacheDir = d
	} else {
		cacheDir = path.Join(os.TempDir(), "gpsearch")
		if !exist(cacheDir) {
			err := os.Mkdir(cacheDir, 0755)
			if err != nil {
				log.Fatalf("Fail to create cache directory %s", cacheDir)
			}
		}
	}
}

// init cacheTimeOut based on env GPSEARCH_CACHETIMEOUT
func init() {
	cacheTimeOut = 2
	ts, ok := os.LookupEnv("GPSEARCH_CACHETIMEOUT")
	if ok {
		ti, err := strconv.Atoi(ts)
		if err != nil {
			cacheTimeOut = ti
		}
	}
}

// getHashName return the sha1 hash string for a given string
func getHashName(name string) string {
	nameHash := fmt.Sprintf("%x", sha1.Sum([]byte(name)))
	return nameHash
}

// exist check if a path exists
func exist(name string) bool {
	_, err := os.Stat(name)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// getCacheName generate a path name based on cache directory and query string
func getCacheName(name string) string {
	fullName := path.Join(cacheDir, getHashName(name))
	return fullName
}

// expire check if the cached data expires
func expire(name string) bool {
	fname := getCacheName(name)
	finfo, err := os.Stat(fname)
	if err != nil {
		return true
	}
	if time.Now().Sub(finfo.ModTime()) > time.Duration(cacheTimeOut)*time.Hour {
		return true
	}
	return false
}

// loadCache load cached data from file
func loadCache(name string, data *[]map[string]interface{}) error {
	var err error
	fname := getCacheName(name)
	if !exist(fname) {
		return fmt.Errorf("Cache file does not exist")
	}

	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	decoder := gob.NewDecoder(f)
	err = decoder.Decode(&data)
	if err != nil {
		return err
	}
	return nil
}

// dumpCache save cached data into file
func dumpCache(name string, data []map[string]interface{}) error {
	var err error
	fname := getCacheName(name)
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := gob.NewEncoder(f)
	err = encoder.Encode(data)
	if err != nil {
		return err
	}
	return nil
}

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
	var err error
	expired := expire(s)
	pkgs := []map[string]interface{}{}
	if !expired {
		err = loadCache(s, &pkgs)
		if err == nil {
			// log.Printf("Successfully load cached data")
			return pkgs, nil
		}
	}

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
	if expired {
		err = dumpCache(s, data["results"])
		if err != nil {
			// log.Printf("Cannot dump query result to cache, ignore silently")
		}
	}
	return data["results"], nil
}

// formatOut print results based on selected fields
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

// sortPkgs sort packages based on a specified field
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
		Name:  "gpsearch",
		Usage: "search golang packages",
		UsageText: `
			search [--sort|-s <field>] [--reverse|-r] [--num|-n <Num. of packages to list>] \
				[[--fields|-f <field1>] [--fields|-f <field2>] [...]] <pattern1> [pattern2] [...]`,
		Description: `
			Search golang packages on godoc.org from CLI

			Fields supported:
				name        : The name
				path        : The import path
				synopsis    : The description
				import_count: Num. of projects using/importing this
				stars       : Num. of github stars
				score       : How well is the document
				fork        : If this is a forked repository

			Notes: Some packages only have few fields with data
		`,
		Version: "1.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "sort",
				Aliases: []string{"s"},
				Value:   "import_count",
				Usage:   "Sort packages based on",
			},
			&cli.BoolFlag{
				Name:    "reverse",
				Aliases: []string{"r"},
				Value:   false,
				Usage:   "Reverse the sort result",
			},
			&cli.Uint64Flag{
				Name:    "num",
				Aliases: []string{"n"},
				Value:   10,
				Usage:   "Num. of packages to list",
			},
			&cli.StringSliceFlag{
				Name:    "fields",
				Aliases: []string{"f"},
				Value:   cli.NewStringSlice("path", "import_count", "synopsis"),
				Usage:   "Package fields to show, speficie multiple values with -f <field1> -f <field2> ...",
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
