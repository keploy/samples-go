# --- Stage 1: Build ---
# Use the official Go image as a builder.
# Using a specific version is a good practice (e.g., 1.24).
# The '-alpine' tag provides a lightweight version of the image.
FROM golang:1.24.2-alpine AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy the go.mod and go.sum files to leverage Docker's build cache.
# This way, dependencies are only re-downloaded when they change.
COPY go.mod ./
RUN go mod download

# Copy the rest of the source code into the container.
COPY . .

# Build the Go application.
# CGO_ENABLED=0 creates a statically linked binary, which is important for
# running in a minimal container like Debian slim.
# -o /app/main specifies the output file name and location.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main .

# --- Stage 2: Final Image ---
# Use a minimal, non-root Debian-based image for the final container.
# 'bookworm-slim' is a good choice for its small size.
FROM debian:bookworm-slim

# Set the working directory for the final stage.
WORKDIR /app

RUN apt-get update && apt-get install -y ca-certificates
# Download the ca.crt file and the setup_ca.sh script
ADD  https://raw.githubusercontent.com/keploy/keploy/refs/heads/main/pkg/core/proxy/tls/asset/ca.crt ca.crt
ADD https://raw.githubusercontent.com/keploy/keploy/refs/heads/main/pkg/core/proxy/tls/asset/setup_ca.sh setup_ca.sh
# Give execute permission to the setup_ca.sh script
RUN chmod +x setup_ca.sh

COPY --from=builder /app/main /app/main

# Expose port 8080 to allow traffic to the server.
EXPOSE 8080

# Run the CA setup script and then run the application server
CMD ["/bin/bash", "-c", "source ./setup_ca.sh && /app/main"]
