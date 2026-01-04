package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	dbFilePath = "restaurants.json"
	fileMutex  = &sync.Mutex{}
)

// DuplicateCheckResponse defines the structure for the duplicate check result from the ML API.
type DuplicateCheckResponse struct {
	IsDuplicate    bool    `json:"is_duplicate"`
	MatchedName    string  `json:"matched_name"`
	SimilarityScore float64 `json:"similarity_score"`
}

// initDB ensures the JSON database file exists.
func initDB(filepath string) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	dbFilePath = filepath
	if _, err := os.Stat(dbFilePath); os.IsNotExist(err) {
		log.Println("Creating database file:", dbFilePath)
		// Create an empty JSON array `[]`
		if err := os.WriteFile(dbFilePath, []byte("[]"), 0644); err != nil {
			log.Fatalf("Failed to create database file: %v", err)
		}
	}
	log.Println("Database file is ready.")
}

// GetAllRestaurants retrieves all restaurants from the JSON file.
func GetAllRestaurants() ([]string, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	data, err := os.ReadFile(dbFilePath)
	if err != nil {
		return nil, err
	}

	var restaurants []string
	if err := json.Unmarshal(data, &restaurants); err != nil {
		return nil, err
	}

	return restaurants, nil
}

// CheckForDuplicate calls the ML API to check for duplicate restaurant names.
func CheckForDuplicate(name string) (*DuplicateCheckResponse, error) {
	restaurants, err := GetAllRestaurants()
	if err != nil {
		return nil, err
	}

	mlApiURL := os.Getenv("ML_API_URL")
	if mlApiURL == "" {
		mlApiURL = "http://localhost:8000/process-message"
	}

	reqBody := map[string]interface{}{
		"task": "check_duplicate",
		"data": map[string]interface{}{
			"new_name":       name,
			"existing_names": restaurants,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(mlApiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Result DuplicateCheckResponse `json:"result"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	return &apiResp.Result, nil
}


// AddRestaurant first checks for duplicates, then adds a new restaurant if none are found.
// It returns an error if a duplicate is found.
func AddRestaurant(name string) (int, *DuplicateCheckResponse, error) {
	duplicateInfo, err := CheckForDuplicate(name)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to check for duplicates: %w", err)
	}

	if duplicateInfo.IsDuplicate {
		return 0, duplicateInfo, nil // Return duplicate info, no error
	}

	// If no duplicate, add the restaurant
	count, err := ForceAddRestaurant(name)
	return count, nil, err
}

// ForceAddRestaurant adds a new restaurant to the JSON file without checking for duplicates.
func ForceAddRestaurant(name string) (int, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	data, err := os.ReadFile(dbFilePath)
	if err != nil {
		return 0, err
	}

	var restaurants []string
	if err := json.Unmarshal(data, &restaurants); err != nil {
		return 0, err
	}

	restaurants = append(restaurants, name)

	newData, err := json.MarshalIndent(restaurants, "", "  ")
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(dbFilePath, newData, 0644); err != nil {
		return 0, err
	}

	return len(restaurants), nil
}


// RemoveRestaurant removes a restaurant from the JSON file.
// It returns the total number of restaurants after removal, and an error if not found.
func RemoveRestaurant(name string) (int, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	data, err := os.ReadFile(dbFilePath)
	if err != nil {
		return 0, err
	}

	var restaurants []string
	if err := json.Unmarshal(data, &restaurants); err != nil {
		return 0, err
	}

	found := false
	newRestaurants := []string{}
	for _, r := range restaurants {
		if r == name {
			found = true
		} else {
			newRestaurants = append(newRestaurants, r)
		}
	}

	if !found {
		return len(restaurants), fmt.Errorf("restaurant '%s' not found", name)
	}

	newData, err := json.MarshalIndent(newRestaurants, "", "  ")
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(dbFilePath, newData, 0644); err != nil {
		return 0, err
	}

	return len(newRestaurants), nil
}
