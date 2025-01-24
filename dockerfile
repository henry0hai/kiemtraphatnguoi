##
# 1) Build Stage
##
FROM golang:1.20-bullseye AS builder

# Install dev libraries for building with Tesseract
RUN apt-get update && apt-get install -y \
    libleptonica-dev \
    libtesseract-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go.mod and go.sum for module dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy all source files
COPY . .

# Build your Go binary, name it "kiemtraphatnguoi" (or "main")
RUN go build -o kiemtraphatnguoi .

##
# 2) Final / Runtime Stage
##
FROM debian:bullseye-slim

# Install the Tesseract runtime and CA certificates
RUN apt-get update && apt-get install -y \
    tesseract-ocr \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy only our final binary from the builder stage
COPY --from=builder /app/kiemtraphatnguoi /usr/local/bin/kiemtraphatnguoi

# If your app listens on port 8080
EXPOSE 8080

# Use the compiled binary as the entrypoint
ENTRYPOINT ["kiemtraphatnguoi"]