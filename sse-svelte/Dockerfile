# Use an official Golang runtime as a parent image
FROM golang:1.20-bookworm

# Set the working directory inside the container
WORKDIR /app

# Copy the local package files to the container's workspace
COPY . .

# Download the modules the build needs.
# proxy.golang.org now and then resets a download mid-transfer ("stream error:
# ... INTERNAL_ERROR"), which fails an un-retried fetch and the whole build.
# Retried: 5 attempts, 5-20s apart, then fail.
RUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done

# Build the Go application
RUN go build -o main .

# Expose port 8080
EXPOSE 3500

# Command to run the Go application
CMD ["./main"]