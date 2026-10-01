FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum* ./
# proxy.golang.org now and then resets a download mid-transfer ("stream error:
# ... INTERNAL_ERROR"), which fails an un-retried fetch and the whole build.
# Retried: 5 attempts, 5-20s apart, then fail.
RUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done

COPY . .
RUN go build -o /bin/api ./cmd/api

FROM alpine:3.22

WORKDIR /app
COPY --from=build /bin/api /app/api

EXPOSE 8080

CMD ["/app/api"]
