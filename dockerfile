# Start with the official Go image for building
FROM golang:1.20 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download and cache Go modules
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go binary
RUN go build -o main .

# Create a smaller runtime image
FROM debian:bullseye-slim

# Set up necessary dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy the compiled binary from the builder
COPY --from=builder /app/main .

# Copy any additional resources if needed (e.g., config files, etc.)
# COPY ./config ./config

# Expose the port your app will run on
EXPOSE 8080

# Set the binary as the entrypoint
ENTRYPOINT ["./main"]