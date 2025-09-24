## === Build Stage ===
FROM golang:1.25 AS build-stage

WORKDIR /app

# Copy go module files first to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy full application source
COPY . .

# Build the binary INSIDE the /app directory
RUN CGO_ENABLED=0 go build -cover -covermode=atomic -coverpkg=./... -o /app/main .

## === Runtime Stage ===
# Use a golang runtime so `go tool cover` is available for Keploy coverage conversion
FROM golang:1.25-alpine

# Create non-root user and install dumb-init
RUN addgroup -S keploy \
    && adduser -S keploy -G keploy -h /home/keploy \
    && mkdir -p /home/keploy/app \
    && chown -R keploy:keploy /home/keploy/app \
    && apk add --no-cache dumb-init su-exec \
    && rm -rf /var/cache/apk/*

WORKDIR /home/keploy/app

COPY --from=build-stage --chown=keploy:keploy /app/. .

CMD ["dumb-init", "su-exec", "keploy:keploy", "/home/keploy/app/main"]

# App port
EXPOSE 8080

# CMD is ignored because entrypoint execs the binary directly
