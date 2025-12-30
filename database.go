package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

var (
	dbFilePath = "restaurants.json"
	fileMutex  = &sync.Mutex{}
)

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

// AddRestaurant adds a new restaurant to the JSON file.
// It returns the total number of restaurants after adding.
func AddRestaurant(name string) (int, error) {
	// We get the lock for the entire read-modify-write operation.
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
