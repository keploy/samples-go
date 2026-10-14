FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /out/dns-strict-resolver .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/dns-strict-resolver /usr/local/bin/dns-strict-resolver
EXPOSE 8086
ENTRYPOINT ["/usr/local/bin/dns-strict-resolver"]
