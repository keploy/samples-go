FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
# proxy.golang.org now and then resets a download mid-transfer ("stream error:
# ... INTERNAL_ERROR"), which fails an un-retried fetch and the whole build.
# Retried: 5 attempts, 5-20s apart, then fail.
RUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done
COPY main.go ./
RUN CGO_ENABLED=0 go build -o proxy-stress-test .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /app/proxy-stress-test /usr/local/bin/proxy-stress-test
ENTRYPOINT ["proxy-stress-test"]
