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
# proxy.golang.org now and then resets a download mid-transfer ("stream error:
# ... INTERNAL_ERROR"), which fails an un-retried fetch and the whole build.
# Retried: 5 attempts, 5-20s apart, then fail.
RUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done

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

# Keploy injects its per-run MITM CA and the trust env vars automatically; no
# certificate needs to be baked in. (ca-certificates gives the base image real
# public roots.)
RUN apt-get update && apt-get install -y ca-certificates

COPY --from=builder /app/main /app/main

# Expose port 8080 to allow traffic to the server.
EXPOSE 8080

CMD ["/app/main"]
