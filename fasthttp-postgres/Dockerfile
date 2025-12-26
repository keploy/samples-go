# Multi-stage build for Go application
FROM golang:1.21-alpine3.18 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fasthttp-postgres .

# Final stage - minimal runtime image
FROM alpine:3.18

# Install ca-certificates for HTTPS if needed
RUN apk --no-cache add ca-certificates

# Create a non-root user and group to run the application
RUN addgroup -S appuser && adduser -S appuser -G appuser

WORKDIR /app

# Copy the binary from builder and set ownership to the non-root user
COPY --from=builder /app/fasthttp-postgres /app/fasthttp-postgres
RUN chown appuser:appuser /app/fasthttp-postgres

# Expose the port your fasthttp application uses (adjust if different)
EXPOSE 8080

# Switch to the non-root user and run the application
USER appuser
CMD ["./fasthttp-postgres"]