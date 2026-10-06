FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
# proxy.golang.org now and then resets a download mid-transfer ("stream error:
# ... INTERNAL_ERROR"), which fails an un-retried fetch and the whole build.
# Retried: 5 attempts, 5-20s apart, then fail.
RUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done
COPY . .
RUN CGO_ENABLED=0 go build -o gin-pulsar .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/gin-pulsar .
EXPOSE 8090
CMD ["./gin-pulsar"]
