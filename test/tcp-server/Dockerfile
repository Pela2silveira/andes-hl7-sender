# Stage 1: Build the Go binary
FROM golang:1.20-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod ./
RUN go mod download

# Copy the Go source code
COPY . .

# Build the Go binary
RUN go build -o tcp-server main.go

# Stage 2: Create a minimal image to run the Go binary
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the Go binary from the builder stage
COPY --from=builder /app/tcp-server .

# Expose the port the TCP server will listen on
EXPOSE 5555

# Run the Go binary
CMD ["./tcp-server"]
