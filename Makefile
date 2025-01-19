.PHONY: build test clean run docker-build docker-run

# Build variables
BINARY_NAME=server
DOCKER_IMAGE=go-ollama-api

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/server

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Clean build files
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

# Run the application
run: build
	./$(BINARY_NAME)

# Build docker image
docker-build:
	docker build -t $(DOCKER_IMAGE) .

# Run docker container
docker-run:
	docker run -p 8080:8080 $(DOCKER_IMAGE)

# Install dependencies
deps:
	$(GOGET) -v ./...

# Format code
fmt:
	$(GOCMD) fmt ./...

# Run linter
lint:
	golangci-lint run

# All (clean, format, build, test)
all: clean fmt build test
