FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd

RUN go build -o /out/mtls-server ./cmd/server && \
    go build -o /out/mtls-client ./cmd/client

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/mtls-server /usr/local/bin/mtls-server
COPY --from=builder /out/mtls-client /usr/local/bin/mtls-client

ENV APP_BIN=mtls-server

ENTRYPOINT ["/bin/sh", "-c", "exec /usr/local/bin/${APP_BIN}"]
