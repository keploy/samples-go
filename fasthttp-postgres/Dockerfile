FROM golang:1.22-alpine AS builder

WORKDIR /app

#install git (needed for go modules somethimes)
RUN apk add --no-cache git

#copy gomod files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

#copy source files
COPY . .

#build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app

# ---------- Runtime stage ----------
FROM alpine:latest

WORKDIR /app

#Copy binary from builder
COPY --from=builder /app/app .

#App port (change if needed)
EXPOSE 8080

CMD ["./app"]
