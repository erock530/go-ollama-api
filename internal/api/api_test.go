package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erock530/go-ollama-api/internal/config"
	"github.com/erock530/go-ollama-api/internal/db"
	"github.com/erock530/go-ollama-api/internal/models"
	"github.com/gorilla/mux"
)

// Ensure MockDB implements db.DBInterface
var _ db.DBInterface = (*MockDB)(nil)

// MockDB implements the necessary database methods for testing
type MockDB struct {
	apiKeys map[string]*models.APIKey
}

func NewMockDB() *MockDB {
	return &MockDB{
		apiKeys: make(map[string]*models.APIKey),
	}
}

func (m *MockDB) GetAPIKey(key string) (*models.APIKey, error) {
	if apiKey, exists := m.apiKeys[key]; exists {
		return apiKey, nil
	}
	return nil, nil
}

func (m *MockDB) UpdateAPIKeyUsage(key string, tokens int) error {
	if apiKey, exists := m.apiKeys[key]; exists {
		apiKey.Tokens = tokens
		apiKey.LastUsed = time.Now()
	}
	return nil
}

func (m *MockDB) LogAPIUsage(key string) error {
	return nil
}

func (m *MockDB) Close() error {
	return nil
}

func setupTestRouter(mockDB db.DBInterface) *mux.Router {
	router := mux.NewRouter()
	cfg := &config.Config{
		Port:      8080,
		OllamaURL: "http://localhost:11434",
	}
	SetupRoutes(router, mockDB, cfg)
	return router
}

func TestHealthCheckHandler(t *testing.T) {
	mockDB := NewMockDB()
	mockDB.apiKeys["valid-key"] = &models.APIKey{
		Key:       "valid-key",
		Active:    true,
		Tokens:    10,
		RateLimit: 10,
		LastUsed:  time.Now(),
	}

	router := setupTestRouter(mockDB)

	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "Valid API Key",
			apiKey:         "valid-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing API Key",
			apiKey:         "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid API Key",
			apiKey:         "invalid-key",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/health?apikey="+tt.apiKey, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response models.APIResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatal("Failed to decode response body")
				}
				if response.Status != "API is healthy" {
					t.Errorf("unexpected response status: got %v want %v",
						response.Status, "API is healthy")
				}
			}
		})
	}
}

// mockOllamaServer creates a test server that mocks Ollama API responses
func mockOllamaServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"response": "mocked response",
		})
	}))
}

func TestRateLimitMiddleware(t *testing.T) {
	// Setup mock Ollama server
	mockServer := mockOllamaServer()
	defer mockServer.Close()

	// Reset rate limits before test
	rateMutex.Lock()
	rateLimits = make(map[string]*RateLimitInfo)
	rateMutex.Unlock()

	mockDB := NewMockDB()
	mockDB.apiKeys["valid-key"] = &models.APIKey{
		Key:       "valid-key",
		Active:    true,
		Tokens:    2,
		RateLimit: 2,
		LastUsed:  time.Now(),
	}
	mockDB.apiKeys["inactive-key"] = &models.APIKey{
		Key:       "inactive-key",
		Active:    false,
		Tokens:    2,
		RateLimit: 2,
		LastUsed:  time.Now(),
	}

	// Create router with mock Ollama URL
	router := mux.NewRouter()
	cfg := &config.Config{
		Port:      8080,
		OllamaURL: mockServer.URL,
	}
	SetupRoutes(router, mockDB, cfg)

	tests := []struct {
		name           string
		apiKey         string
		numRequests    int
		expectedStatus []int
	}{
		{
			name:           "Within Rate Limit",
			apiKey:         "valid-key",
			numRequests:    1,
			expectedStatus: []int{http.StatusOK},
		},
		{
			name:           "Exceed Rate Limit",
			apiKey:         "valid-key",
			numRequests:    3,
			expectedStatus: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests},
		},
		{
			name:           "Inactive API Key",
			apiKey:         "inactive-key",
			numRequests:    1,
			expectedStatus: []int{http.StatusForbidden},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset rate limits and API key state before each test case
			rateMutex.Lock()
			rateLimits = make(map[string]*RateLimitInfo)
			rateMutex.Unlock()

			// Reset API key tokens
			if tt.apiKey == "valid-key" {
				mockDB.apiKeys["valid-key"] = &models.APIKey{
					Key:       "valid-key",
					Active:    true,
					Tokens:    2,
					RateLimit: 2,
					LastUsed:  time.Now(),
				}
			}
			for i := 0; i < tt.numRequests; i++ {
				// Include model and prompt in request body for generate endpoint
				body := map[string]interface{}{
					"apikey": tt.apiKey,
					"model":  "test-model",
					"prompt": "test prompt",
				}
				jsonBody, _ := json.Marshal(body)
				req, err := http.NewRequest("POST", "/generate", bytes.NewBuffer(jsonBody))
				if err != nil {
					t.Fatal(err)
				}

				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)

				if status := rr.Code; status != tt.expectedStatus[min(i, len(tt.expectedStatus)-1)] {
					t.Errorf("request %d: handler returned wrong status code: got %v want %v",
						i+1, status, tt.expectedStatus[min(i, len(tt.expectedStatus)-1)])
				}
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
