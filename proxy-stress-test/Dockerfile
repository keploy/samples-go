FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -o proxy-stress-test .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /app/proxy-stress-test /usr/local/bin/proxy-stress-test
ENTRYPOINT ["proxy-stress-test"]
