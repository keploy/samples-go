# Stage 1: The 'builder' stage to compile the Go application
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY *.go ./



COPY go_freeze_time_arm64 /lib/keploy/go_freeze_time_arm64

# Set suitable permissions
RUN chmod +x /lib/keploy/go_freeze_time_arm64

# Run the binary to set up the time freezing environment
RUN /lib/keploy/go_freeze_time_arm64


# Build the Go application into a static binary
# CGO_ENABLED=0 is important for creating a static binary that can run in a minimal image
RUN CGO_ENABLED=0 GOOS=linux go build -tags=faketime -o /main .

# Stage 2: The 'final' stage to create the minimal production image
FROM alpine:latest

# Set the working directory
WORKDIR /

# Copy the compiled binary from the 'builder' stage
COPY --from=builder /main /main

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["/main"]