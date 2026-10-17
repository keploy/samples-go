# Use the Go 1.19 base image with the bullseye tag
FROM golang:1.20-bookworm


RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* 

# Set the working directory inside the container
WORKDIR /app

COPY go.mod /app/
COPY go.sum /app/

# proxy.golang.org now and then resets a download mid-transfer ("stream error:
# ... INTERNAL_ERROR"), which fails an un-retried fetch and the whole build.
# Retried: 5 attempts, 5-20s apart, then fail.
RUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done

# Copy the contents from the local directory to the working directory in the container
COPY . /app

RUN go build -o app .

EXPOSE 8080

# Run the application server using "go run main.go"
CMD ["./app"]