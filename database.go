package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite" // The pure Go SQLite driver
)

// InitDB initializes and returns a connection to the database.
func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Create the table if it doesn't exist
	createTableSQL := `CREATE TABLE IF NOT EXISTS restaurants (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"name" TEXT
	);`

	statement, err := db.Prepare(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to prepare table creation statement: %v", err)
	}
	statement.Exec()

	log.Println("Database initialized and table created successfully.")
	return db
}

// AddRestaurant adds a new restaurant to the database.
func AddRestaurant(db *sql.DB, name string) (int64, error) {
	insertSQL := `INSERT INTO restaurants(name) VALUES (?)`
	statement, err := db.Prepare(insertSQL)
	if err != nil {
		return 0, err
	}
	defer statement.Close()

	result, err := statement.Exec(name)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetAllRestaurants retrieves all restaurants from the database.
func GetAllRestaurants(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM restaurants")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var restaurants []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		restaurants = append(restaurants, name)
	}

	return restaurants, nil
}
