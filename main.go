package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// API_URL is the url of the api the sunrise/sunset data comes from
	API_URL = "https://api.sunrise-sunset.org/json"
	// APP_CACHE_DIR directory of the where the CACHE_FILE will be saved.
	APP_CACHE_DIR = "asadesuka"
	// CACHE_FILE is the file name for the cache file.
	CACHE_FILE = "data.json"
)

type results struct {
	// Sunrise is a time.RFC3339 format date string.
	Sunrise string `json:"sunrise"`
	// Sunset is a time.RFC3339 format date string.
	Sunset string `json:"sunset"`
}

type apiResp struct {
	// Results is a struct containing date strings for sunrise and sunset.
	Results results `json:"results"`
	// Status is the status text of the data
	Status string `json:"status"`
	// TzId is the tzid of the data.
	TzId string `json:"tzid"`
}

// isCacheValid will verify if the cache is still valid or not.
// It will return false if there is no cache.
func isCacheValid() (bool, error) {
	timeFormat := "2006-01-02"
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return false, err
	}

	stat, err := os.Stat(filepath.Join(
		cacheDir,
		APP_CACHE_DIR,
		CACHE_FILE,
	))

	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Join(cacheDir, APP_CACHE_DIR), 0750)
		if err != nil {
			return false, err
		}

		return false, nil
	}

	// only compare the datestring so that when it is the next day, it just fetches a new data
	return stat.ModTime().Format(timeFormat) == time.Now().Format(timeFormat), nil
}

// loadCache will try to load a local cache in the user's cache directory.
func loadCache() (*apiResp, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filepath.Join(
		cacheDir,
		APP_CACHE_DIR,
		CACHE_FILE,
	))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data apiResp
	err = json.NewDecoder(file).Decode(&data)
	return &data, err
}

// fetchSunData gets the new sunrise and sunset data.
func fetchSunData(url string) (*apiResp, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data apiResp
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(data.Status) != "ok" {
		return nil, errors.New("Failed to fetch new sun data.")
	}

	err = saveCache(&data)
	if err != nil {
		fmt.Printf("Error saving sun data: %s\n", err.Error())
	}

	return &data, nil
}

// saveCache saves the given sunrise/sunset data into cache directory.
func saveCache(data *apiResp) error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(
		cacheDir,
		APP_CACHE_DIR,
		CACHE_FILE,
	))
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

func main() {
	lat := os.Getenv("ASA_LAT")
	lng := os.Getenv("ASA_LNG")
	// for better accuracy, export this variable. Supported tzid https://www.php.net/manual/en/timezones.php
	tzid := os.Getenv("ASA_TZID")
	if tzid == "" {
		// use default to UTC
		tzid = "UTC"
	}
	// escape the query parameters to safely use them in an URL
	params := url.Values{}
	params.Add("lat", lat)
	params.Add("lng", lng)
	params.Add("tzid", tzid)
	params.Add("formatted", "0")
	apiUrl := fmt.Sprintf("%s?%s", API_URL, params.Encode())

	// load up the data
	var cache *apiResp
	isValid, err := isCacheValid()
	if err != nil {
		log.Fatal(err)
	}

	if isValid {
		cache, err = loadCache()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cache, err = fetchSunData(apiUrl)
		if err != nil {
			log.Fatal(err)
		}
	}

	now := time.Now()
	sunrise, err := time.Parse(time.RFC3339, cache.Results.Sunrise)
	if err != nil {
		log.Fatal(err)
	}
	sunset, err := time.Parse(time.RFC3339, cache.Results.Sunset)
	if err != nil {
		log.Fatal(err)
	}

	if now.After(sunrise) && now.Before(sunset) {
		fmt.Println("true")
	} else {
		fmt.Println("false")
	}
}
