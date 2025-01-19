# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -o server ./cmd/server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache sqlite-libs ca-certificates

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Expose port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV OLLAMA_URL=http://host.docker.internal:11434

# Run the application
CMD ["./server"]
