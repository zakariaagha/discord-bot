package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// PendingConfirmation holds the state for a pending user confirmation.
type PendingConfirmation struct {
	NewName     string
	MatchedName string
}

var (
	// pendingConfirmations stores pending duplicate confirmations, keyed by channel ID.
	pendingConfirmations      = make(map[string]PendingConfirmation)
	pendingConfirmationsMutex sync.Mutex
)

// Handler is now an empty struct as it doesn't need to hold a database connection.
type Handler struct{}

// HandleMessage is a method of the Handler struct that handles incoming messages.
func (h *Handler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check for a pending confirmation first.
	pendingConfirmationsMutex.Lock()
	if confirmation, ok := pendingConfirmations[m.ChannelID]; ok {
		switch strings.ToLower(m.Content) {
		case "!yes":
			delete(pendingConfirmations, m.ChannelID)
			pendingConfirmationsMutex.Unlock()
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Okay, I will not add '%s'.", confirmation.NewName))
			return
		case "!no":
			delete(pendingConfirmations, m.ChannelID)
			pendingConfirmationsMutex.Unlock()
			count, err := ForceAddRestaurant(confirmation.NewName)
			if err != nil {
				log.Printf("Failed to force-add restaurant: %v", err)
				s.ChannelMessageSend(m.ChannelID, "Something went wrong while adding the restaurant.")
			} else {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Okay, I've added '%s'. Total count: %d.", confirmation.NewName, count))
			}
			return
		default:
			s.ChannelMessageSend(m.ChannelID, "Please respond with `!yes` or `!no`.")
			pendingConfirmationsMutex.Unlock()
			return
		}
	}
	pendingConfirmationsMutex.Unlock()


	if m.Content == "!ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
		return
	}

	if m.Content == "!list" {
		restaurants, err := GetAllRestaurants()
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

	if m.Content == "!ml" {
		err := HealthCheckMLAPI()
		if err != nil {
			log.Printf("ML API health check failed: %v", err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("ML API health check failed: %v", err))
		} else {
			s.ChannelMessageSend(m.ChannelID, "Milinda, present!")
		}
		return
	}

	if strings.HasPrefix(m.Content, "!remove \"") && strings.HasSuffix(m.Content, "\"") {
		restaurantName := strings.TrimSuffix(strings.TrimPrefix(m.Content, "!remove \""), "\"")
		if restaurantName == "" {
			s.ChannelMessageSend(m.ChannelID, "Please provide a restaurant name to remove.")
			return
		}

		count, err := RemoveRestaurant(restaurantName)
		if err != nil {
			log.Printf("Failed to remove restaurant: %v", err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to remove restaurant: %v", err))
			return
		}

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Removed restaurant \"%s\". Total count: %d.", restaurantName, count))
		return
	}

	if strings.HasPrefix(m.Content, "!add \"") && strings.HasSuffix(m.Content, "\"") {
		restaurantName := strings.TrimSuffix(strings.TrimPrefix(m.Content, "!add \""), "\"")
		if restaurantName == "" {
			s.ChannelMessageSend(m.ChannelID, "Please provide a restaurant name.")
			return
		}

		count, duplicateInfo, err := AddRestaurant(restaurantName)
		if err != nil {
			log.Printf("Error adding restaurant: %v", err)
			s.ChannelMessageSend(m.ChannelID, "An error occurred while trying to add the restaurant.")
			return
		}

		if duplicateInfo != nil {
			// A potential duplicate was found, ask for confirmation.
			pendingConfirmationsMutex.Lock()
			pendingConfirmations[m.ChannelID] = PendingConfirmation{
				NewName:     restaurantName,
				MatchedName: duplicateInfo.MatchedName,
			}
			pendingConfirmationsMutex.Unlock()
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Is '%s' the same as '%s'? (Please respond with `!yes` or `!no`)", restaurantName, duplicateInfo.MatchedName))
			return
		}

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Added restaurant \"%s\". Total count: %d.", restaurantName, count))
	}
}

// HealthCheckMLAPI checks the status of the ML API.
func HealthCheckMLAPI() error {
	mlApiURL := os.Getenv("ML_API_URL")
	if mlApiURL == "" {
		mlApiURL = "http://localhost:8000/"
	}

	resp, err := http.Get(mlApiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ML API returned non-200 status: %s", resp.Status)
	}

	return nil
}

func main() {
	// Load .env file if it exists, but don't fail if it doesn't.
	godotenv.Load()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("Error: DISCORD_TOKEN environment variable not set.")
	}

	// Get the database path from the environment variable, with a default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get user home directory: %v", err)
		}
		dbPath = filepath.Join(homeDir, "restaurants.json")
	}

	// Initialize the database file
	initDB(dbPath)

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Specify the necessary intents.
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	// Create a new handler
	h := &Handler{}

	dg.AddHandler(h.HandleMessage)

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