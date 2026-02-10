# Stage 1: The 'builder' stage
FROM golang:1.24-bookworm AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY *.go ./

# Build the Go application
# You can keep CGO_ENABLED=0 for a static binary, or use CGO_ENABLED=1. 
# Since we are switching the final image to Debian as well, CGO_ENABLED=0 is safest for portability.
RUN CGO_ENABLED=0 GOOS=linux go build -o /main .

FROM debian:bookworm-slim

# Set the working directory
WORKDIR /

# Copy the compiled binary from the 'builder' stage
COPY --from=builder /main /main

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["/main"]