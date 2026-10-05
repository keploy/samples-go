# Multi-stage build for optimized image size
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

# Final stage - runtime image
FROM alpine:3.18
WORKDIR /app
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/app .
EXPOSE 3000
CMD ["./app"]
