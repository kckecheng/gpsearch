// Package cache save and load data into/from a local file
package cache

import (
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"time"
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

// Expired check if the cached data expires
func Expired(name string) bool {
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

// Load cached data from file
func Load(name string, data *[]map[string]interface{}) error {
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

// Save cached data into file
func Save(name string, data []map[string]interface{}) error {
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
