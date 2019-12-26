// Package search godoc.org
package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/kckecheng/gpsearch/cache"
)

// Const definition
const (
	goDocEP = "https://api.godoc.org/search?q=" // godoc.org API endpoint
	ErrExit = 1                                 // ErrExit code for abnormal exist
)

// Search definition
type Search struct {
	qstr string
	pkgs []map[string]interface{}
}

// reverseBool reverse bool
func reverseBool(orig, reverse bool) bool {
	if reverse {
		return !orig
	}
	return orig
}

// NewSearch search qstr against godoc.org
func NewSearch(qstr string) (*Search, error) {
	var err error
	expired := cache.Expired(qstr)
	pkgs := []map[string]interface{}{}
	if !expired {
		err = cache.Load(qstr, &pkgs)
		if err == nil {
			// log.Printf("Successfully load cached data")
			return &Search{qstr, pkgs}, nil
		}
	}

	url := goDocEP + fmt.Sprintf("%s", qstr)
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
		err = cache.Save(qstr, data["results"])
		if err != nil {
			// log.Printf("Cannot dump query result to cache, ignore silently")
		}
	}
	return &Search{qstr, data["results"]}, nil
}

// FormatOutN print N num. of pkgs based on selected fields
func (s *Search) FormatOutN(fields []string, n int) {
	length := len(s.pkgs)
	if n > length {
		fmt.Printf("Only %d(<%d) packages exist, list them all\n", length, n)
		n = length
	}
	if n >= 0 && n <= length {
		pkgs := s.pkgs
		for i := 0; i < n; i++ {
			for _, field := range fields {
				v, ok := pkgs[i][field]
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
}

// FormatOut print all pkgs based on selected fields
func (s *Search) FormatOut(fields []string) {
	n := len(s.pkgs)
	s.FormatOutN(fields, n)
}

// Sort sort packages based on a specified field
func (s *Search) Sort(field string, reverse bool) {
	greater := func(i, j int) bool {
		v1, ok1 := s.pkgs[i][field]
		v2, ok2 := s.pkgs[j][field]
		if !ok1 && !ok2 {
			return reverseBool(true, reverse)
		} else if ok1 && !ok2 {
			return reverseBool(true, reverse)
		} else if !ok1 && ok2 {
			return reverseBool(false, reverse)
		}

		switch v1.(type) {
		case string:
			// Reverse order for string: a goes first then b, etc.
			return reverseBool((v1.(string) < v2.(string)), reverse)
		case float32:
			return reverseBool((v1.(float32) > v2.(float32)), reverse)
		case float64:
			return reverseBool((v1.(float64) > v2.(float64)), reverse)
		case int32:
			return reverseBool((v1.(int32) > v2.(int32)), reverse)
		case int64:
			return reverseBool((v1.(int64) > v2.(int64)), reverse)
		case bool:
			if v1.(bool) || (!v1.(bool) && !v2.(bool)) {
				return reverseBool(true, reverse)
			}
			return reverseBool(false, reverse)
		}

		return reverseBool(false, reverse)
	}
	sort.SliceStable(s.pkgs, greater)
}
