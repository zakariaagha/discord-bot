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

// ProcessMessageRequest defines the structure for the request to the ML API.
type ProcessMessageRequest struct {
	Task string                 `json:"task"`
	Data map[string]interface{} `json:"data"`
}

// ProcessMessageResponse defines the structure for the response from the ML API.
type ProcessMessageResponse struct {
	Task   string                 `json:"task"`
	Result map[string]interface{} `json:"result"`
}

// initDB ensures the JSON database file exists.
func initDB(filepath string) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	dbFilePath = filepath
	if _, err := os.Stat(dbFilePath); os.IsNotExist(err) {
		log.Println("Creating database file:", dbFilePath)
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

// processMessage calls the ML API to perform a task.
func processMessage(task string, data map[string]interface{}) (*ProcessMessageResponse, error) {
	mlApiURL := os.Getenv("ML_API_URL")
	if mlApiURL == "" {
		mlApiURL = "http://localhost:8000/process-message"
	}

	reqBody := ProcessMessageRequest{
		Task: task,
		Data: data,
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

	var apiResp ProcessMessageResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}

// AddRestaurant adds a new restaurant to the JSON file after checking for duplicates.
// It returns the total number of restaurants after adding, and an error if a duplicate is found.
func AddRestaurant(name string) (int, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	restaurants, err := GetAllRestaurants()
	if err != nil {
		return 0, err
	}

	// Check for duplicates
	resp, err := processMessage("check_duplicate", map[string]interface{}{
		"new_name":       name,
		"existing_names": restaurants,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to check for duplicates: %w", err)
	}

	if isDuplicate, ok := resp.Result["is_duplicate"].(bool); ok && isDuplicate {
		matchedName, _ := resp.Result["matched_name"].(string)
		return 0, fmt.Errorf("restaurant '%s' is a duplicate of '%s'", name, matchedName)
	}

	// Add the restaurant if no duplicate is found
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
