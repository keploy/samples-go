# Multi-stage build for optimized image size
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

# Final stage - runtime image
FROM alpine:3.18

WORKDIR /app

# Install ca-certificates for HTTPS (if needed)
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/app .

# Expose the application port
EXPOSE 8080

# Health check to verify container is running
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/books || exit 1

# Run the application
CMD ["./app"]