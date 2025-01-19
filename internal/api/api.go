package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/erock530/go-ollama-api/internal/config"
	"github.com/erock530/go-ollama-api/internal/db"
	"github.com/erock530/go-ollama-api/internal/models"

	"github.com/gorilla/mux"
)

var (
	rateLimits = make(map[string]*RateLimitInfo)
	rateMutex  sync.RWMutex
)

// RateLimitInfo tracks rate limiting information for an API key
type RateLimitInfo struct {
	Tokens    int
	LastUsed  time.Time
	RateLimit int
}

// SetupRoutes configures the API routes
func SetupRoutes(r *mux.Router, db db.DBInterface, cfg *config.Config) {
	r.Use(func(next http.Handler) http.Handler {
		return rateLimitMiddleware(next, db)
	})

	r.HandleFunc("/health", healthCheckHandler(db)).Methods("GET")
	r.HandleFunc("/generate", generateHandler(db, cfg)).Methods("POST")
}

// rateLimitMiddleware handles API key validation and rate limiting
func rateLimitMiddleware(next http.Handler, db db.DBInterface) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health check endpoint
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		var req struct {
			APIKey string `json:"apikey"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		// Reset body for next handler
		r.Body = io.NopCloser(bytes.NewBuffer([]byte(`{"apikey":"` + req.APIKey + `"}`)))

		if req.APIKey == "" {
			http.Error(w, "API key is required", http.StatusBadRequest)
			return
		}

		apiKey, err := db.GetAPIKey(req.APIKey)
		if err != nil {
			log.Printf("Error checking API key: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if apiKey == nil {
			http.Error(w, "Invalid API key", http.StatusForbidden)
			return
		}
		if !apiKey.Active {
			http.Error(w, "API key is deactivated", http.StatusForbidden)
			return
		}

		rateMutex.Lock()
		info, exists := rateLimits[req.APIKey]
		if !exists {
			info = &RateLimitInfo{
				Tokens:    apiKey.Tokens,
				LastUsed:  apiKey.LastUsed,
				RateLimit: apiKey.RateLimit,
			}
			rateLimits[req.APIKey] = info
		}

		currentTime := time.Now()
		if currentTime.Sub(info.LastUsed) >= time.Minute {
			info.Tokens = info.RateLimit
		}

		if info.Tokens > 0 {
			info.Tokens--
			info.LastUsed = currentTime
			rateMutex.Unlock()

			if err := db.UpdateAPIKeyUsage(req.APIKey, info.Tokens); err != nil {
				log.Printf("Error updating API key usage: %v", err)
			}

			next.ServeHTTP(w, r)
		} else {
			rateMutex.Unlock()
			http.Error(w, "Rate limit exceeded. Try again later.", http.StatusTooManyRequests)
		}
	})
}

// healthCheckHandler handles the health check endpoint
func healthCheckHandler(db db.DBInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.URL.Query().Get("apikey")
		if apiKey == "" {
			http.Error(w, "API key is required", http.StatusBadRequest)
			return
		}

		key, err := db.GetAPIKey(apiKey)
		if err != nil {
			log.Printf("Error checking API key: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if key == nil {
			http.Error(w, "Invalid API key", http.StatusForbidden)
			return
		}

		response := models.APIResponse{
			Status:    "API is healthy",
			Timestamp: time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// generateHandler handles the generate endpoint that proxies to Ollama
func generateHandler(db db.DBInterface, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Create request to Ollama API
		ollamaReq := struct {
			Model  string   `json:"model"`
			Prompt string   `json:"prompt"`
			Stream bool     `json:"stream"`
			Images []string `json:"images,omitempty"`
			Raw    bool     `json:"raw"`
		}{
			Model:  req.Model,
			Prompt: req.Prompt,
			Stream: req.Stream,
			Images: req.Images,
			Raw:    req.Raw,
		}

		ollamaBody, err := json.Marshal(ollamaReq)
		if err != nil {
			http.Error(w, "Error preparing request", http.StatusInternalServerError)
			return
		}

		ollamaResp, err := http.Post(cfg.OllamaURL+"/api/generate", "application/json", bytes.NewBuffer(ollamaBody))
		if err != nil {
			log.Printf("Error making request to Ollama API: %v", err)
			http.Error(w, "Error making request to Ollama API", http.StatusInternalServerError)
			return
		}
		defer ollamaResp.Body.Close()

		// Log API usage
		if err := db.LogAPIUsage(req.APIKey); err != nil {
			log.Printf("Error logging API usage: %v", err)
		}

		// Forward Ollama response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(ollamaResp.StatusCode)
		io.Copy(w, ollamaResp.Body)
	}
}
