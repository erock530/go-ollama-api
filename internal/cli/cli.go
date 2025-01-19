package cli

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/erock530/go-ollama-api/internal/db"
)

// CLI represents the command-line interface
type CLI struct {
	db *db.DB
}

// NewCLI creates a new CLI instance
func NewCLI(db *db.DB) *CLI {
	return &CLI{db: db}
}

// HandleCommand processes a CLI command
func (c *CLI) HandleCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "generatekey":
		c.generateKey()
	case "generatekeys":
		if len(args) > 0 {
			if count, err := strconv.Atoi(args[0]); err == nil {
				c.generateKeys(count)
			} else {
				fmt.Println("Invalid number of keys")
			}
		} else {
			fmt.Println("Please specify the number of keys to generate")
		}
	case "listkeys":
		c.listKeys()
	case "removekey":
		if len(args) > 0 {
			c.removeKey(args[0])
		} else {
			fmt.Println("Please specify the API key to remove")
		}
	case "addwebhook":
		if len(args) > 0 {
			c.addWebhook(args[0])
		} else {
			fmt.Println("Please specify the webhook URL")
		}
	case "deletewebhook":
		if len(args) > 0 {
			if id, err := strconv.ParseInt(args[0], 10, 64); err == nil {
				c.deleteWebhook(id)
			} else {
				fmt.Println("Invalid webhook ID")
			}
		} else {
			fmt.Println("Please specify the webhook ID")
		}
	case "listwebhooks":
		c.listWebhooks()
	case "help":
		c.printHelp()
	default:
		fmt.Println("Unknown command. Type 'help' for available commands.")
	}
}

// generateKey generates a single API key
func (c *CLI) generateKey() {
	key, err := generateRandomKey()
	if err != nil {
		log.Printf("Error generating key: %v", err)
		return
	}

	_, err = c.db.Exec(`
		INSERT INTO apiKeys (key, rate_limit) 
		VALUES (?, 10)`,
		key,
	)
	if err != nil {
		log.Printf("Error saving API key: %v", err)
		return
	}

	fmt.Printf("Generated API key: %s\n", key)
}

// generateKeys generates multiple API keys
func (c *CLI) generateKeys(count int) {
	for i := 0; i < count; i++ {
		c.generateKey()
	}
}

// listKeys lists all API keys
func (c *CLI) listKeys() {
	rows, err := c.db.Query(`
		SELECT key, created_at, last_used, tokens, rate_limit, active, description 
		FROM apiKeys
	`)
	if err != nil {
		log.Printf("Error listing API keys: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\nAPI Keys:")
	fmt.Println("----------------------------------------")
	for rows.Next() {
		var key string
		var description sql.NullString
		var createdAt, lastUsed string
		var tokens, rateLimit int
		var active bool
		if err := rows.Scan(&key, &createdAt, &lastUsed, &tokens, &rateLimit, &active, &description); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("Key: %s\n", key)
		fmt.Printf("Created: %s\n", createdAt)
		fmt.Printf("Last Used: %s\n", lastUsed)
		fmt.Printf("Tokens: %d\n", tokens)
		fmt.Printf("Rate Limit: %d\n", rateLimit)
		fmt.Printf("Active: %v\n", active)
		if description.Valid {
			fmt.Printf("Description: %s\n", description.String)
		}
		fmt.Println("----------------------------------------")
	}
}

// removeKey removes an API key
func (c *CLI) removeKey(key string) {
	result, err := c.db.Exec("DELETE FROM apiKeys WHERE key = ?", key)
	if err != nil {
		log.Printf("Error removing API key: %v", err)
		return
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		fmt.Println("No API key found with that value")
	} else {
		fmt.Println("API key removed successfully")
	}
}

// addWebhook adds a new webhook
func (c *CLI) addWebhook(url string) {
	_, err := c.db.Exec("INSERT INTO webhooks (url) VALUES (?)", url)
	if err != nil {
		log.Printf("Error adding webhook: %v", err)
		return
	}
	fmt.Println("Webhook added successfully")
}

// deleteWebhook deletes a webhook
func (c *CLI) deleteWebhook(id int64) {
	result, err := c.db.Exec("DELETE FROM webhooks WHERE id = ?", id)
	if err != nil {
		log.Printf("Error deleting webhook: %v", err)
		return
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		fmt.Println("No webhook found with that ID")
	} else {
		fmt.Println("Webhook deleted successfully")
	}
}

// listWebhooks lists all webhooks
func (c *CLI) listWebhooks() {
	rows, err := c.db.Query("SELECT id, url FROM webhooks")
	if err != nil {
		log.Printf("Error listing webhooks: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\nWebhooks:")
	fmt.Println("----------------------------------------")
	for rows.Next() {
		var id int64
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("ID: %d\n", id)
		fmt.Printf("URL: %s\n", url)
		fmt.Println("----------------------------------------")
	}
}

// printHelp prints available commands
func (c *CLI) printHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  generatekey           - Generate a single API key")
	fmt.Println("  generatekeys <count>  - Generate multiple API keys")
	fmt.Println("  listkeys             - List all API keys")
	fmt.Println("  removekey <key>      - Remove an API key")
	fmt.Println("  addwebhook <url>     - Add a webhook URL")
	fmt.Println("  deletewebhook <id>   - Delete a webhook")
	fmt.Println("  listwebhooks         - List all webhooks")
	fmt.Println("  help                 - Show this help message")
	fmt.Println("  exit                 - Exit the program")
}

// generateRandomKey generates a random API key
func generateRandomKey() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
