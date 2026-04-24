FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/pcap-validator .

FROM alpine:3.21

RUN adduser -D -u 10001 appuser
USER appuser

WORKDIR /app
COPY --from=build /out/pcap-validator /app/pcap-validator

EXPOSE 8080 8443
ENTRYPOINT ["/app/pcap-validator"]
