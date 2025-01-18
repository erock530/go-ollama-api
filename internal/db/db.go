package db

import (
	"database/sql"
	"time"

	"github.com/erock530/go-ollama-api/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQL database connection
type DB struct {
	*sql.DB
}

// InitDB initializes the database connection and creates tables
func InitDB() (*DB, error) {
	db, err := sql.Open("sqlite3", "./apiKeys.db")
	if err != nil {
		return nil, err
	}

	if err := createTables(db); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

// createTables creates the necessary database tables if they don't exist
func createTables(db *sql.DB) error {
	// Create apiKeys table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS apiKeys (
			key TEXT PRIMARY KEY,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			tokens INTEGER DEFAULT 10,
			rate_limit INTEGER DEFAULT 10,
			active INTEGER DEFAULT 1,
			description TEXT
		)
	`)
	if err != nil {
		return err
	}

	// Create apiUsage table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS apiUsage (
			key TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create webhooks table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS webhooks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

// GetAPIKey retrieves an API key from the database
func (db *DB) GetAPIKey(key string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := db.QueryRow(`
		SELECT key, created_at, last_used, tokens, rate_limit, active, description 
		FROM apiKeys WHERE key = ?`, key).Scan(
		&apiKey.Key,
		&apiKey.CreatedAt,
		&apiKey.LastUsed,
		&apiKey.Tokens,
		&apiKey.RateLimit,
		&apiKey.Active,
		&apiKey.Description,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// UpdateAPIKeyUsage updates the usage information for an API key
func (db *DB) UpdateAPIKeyUsage(key string, tokens int) error {
	_, err := db.Exec(`
		UPDATE apiKeys 
		SET tokens = ?, last_used = ? 
		WHERE key = ?`,
		tokens,
		time.Now(),
		key,
	)
	return err
}

// LogAPIUsage logs an API usage event
func (db *DB) LogAPIUsage(key string) error {
	_, err := db.Exec(`INSERT INTO apiUsage (key) VALUES (?)`, key)
	return err
}

// GetWebhooks retrieves all webhook URLs
func (db *DB) GetWebhooks() ([]models.Webhook, error) {
	rows, err := db.Query("SELECT id, url FROM webhooks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []models.Webhook
	for rows.Next() {
		var webhook models.Webhook
		if err := rows.Scan(&webhook.ID, &webhook.URL); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook)
	}
	return webhooks, nil
}

// AddWebhook adds a new webhook URL
func (db *DB) AddWebhook(url string) error {
	_, err := db.Exec("INSERT INTO webhooks (url) VALUES (?)", url)
	return err
}

// DeleteWebhook deletes a webhook by ID
func (db *DB) DeleteWebhook(id int64) error {
	_, err := db.Exec("DELETE FROM webhooks WHERE id = ?", id)
	return err
}
