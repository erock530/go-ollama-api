package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/erock530/go-ollama-api/internal/api"
	"github.com/erock530/go-ollama-api/internal/cli"
	"github.com/erock530/go-ollama-api/internal/config"
	"github.com/erock530/go-ollama-api/internal/db"

	"github.com/gorilla/mux"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "Port to run the server on")
	ollamaURL := flag.String("ollama-url", "http://127.0.0.1:11434", "URL of the Ollama server")
	flag.Parse()

	// Initialize configuration
	cfg := &config.Config{
		Port:      *port,
		OllamaURL: *ollamaURL,
	}

	// Initialize database
	database, err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create router
	router := mux.NewRouter()

	// Initialize API handlers
	api.SetupRoutes(router, database, cfg)

	// Create server with graceful shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	// Initialize CLI
	cli := cli.NewCLI(database)

	// Channel for shutdown signals
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Start CLI in a goroutine
	go func() {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("CLI ready. Type 'help' for available commands.")

		for {
			fmt.Print("> ")
			input, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("Error reading input: %v", err)
				continue
			}

			input = strings.TrimSpace(input)
			if input == "exit" {
				quit <- syscall.SIGTERM
				return
			}

			cli.HandleCommand(input)
		}
	}()

	// Wait for shutdown signal
	<-quit
	log.Println("Server is shutting down...")

	// Gracefully shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	close(done)
	log.Println("Server stopped")
}
