package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// Handler is a struct that holds the database connection.
type Handler struct {
	DB *sql.DB
}

// MessageCreate is a method of the Handler struct that handles incoming messages.
func (h *Handler) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
		return
	}

	if m.Content == "!list" {
		restaurants, err := GetAllRestaurants(h.DB)
		if err != nil {
			log.Printf("Failed to get restaurants: %v", err)
			s.ChannelMessageSend(m.ChannelID, "Failed to get restaurants.")
			return
		}

		if len(restaurants) == 0 {
			s.ChannelMessageSend(m.ChannelID, "No restaurants found.")
			return
		}

		s.ChannelMessageSend(m.ChannelID, "Restaurants:\n- "+strings.Join(restaurants, "\n- "))
		return
	}

	if strings.HasPrefix(m.Content, "add \"") && strings.HasSuffix(m.Content, "\"") {
		restaurantName := strings.TrimSuffix(strings.TrimPrefix(m.Content, "add \""), "\"")
		if restaurantName == "" {
			s.ChannelMessageSend(m.ChannelID, "Please provide a restaurant name.")
			return
		}

		id, err := AddRestaurant(h.DB, restaurantName)
		if err != nil {
			log.Printf("Failed to add restaurant: %v", err)
			s.ChannelMessageSend(m.ChannelID, "Failed to add restaurant.")
			return
		}

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Added restaurant \"%s\" with ID %d.", restaurantName, id))
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		fmt.Println("Error: DISCORD_TOKEN environment variable not set.")
		return
	}

	// Get the database path from the environment variable, with a default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get user home directory: %v", err)
		}
		dbPath = filepath.Join(homeDir, "discord-bot.db")
	}

	// Initialize the database
	db := InitDB(dbPath)
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Specify the necessary intents.
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	// Create a new handler with the database connection
	h := &Handler{DB: db}

	dg.AddHandler(h.MessageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}