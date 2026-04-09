FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -o repro-app .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /app/repro-app /usr/local/bin/repro-app
ENTRYPOINT ["repro-app"]
