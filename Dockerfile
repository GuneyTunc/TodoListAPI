# Builder stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to cache dependencies
COPY go.mod .
COPY go.sum .

# Download Go modules
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
# CGO_ENABLED=0 for static linking, GOOS=linux for Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the compiled Go application from the builder stage
COPY --from=builder /app/main .


# Expose the application port
EXPOSE 8080

# Command to run the application
CMD ["./main"]
