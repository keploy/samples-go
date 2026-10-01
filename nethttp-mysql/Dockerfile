FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod go.sum* ./
# proxy.golang.org now and then resets a download mid-transfer ("stream error:
# ... INTERNAL_ERROR"), which fails an un-retried fetch and the whole build.
# Retried: 5 attempts, 5-20s apart, then fail.
RUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done
COPY . .
RUN n=0; until go mod tidy; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod tidy failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM alpine:3.19
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=build /app/app /app/app
EXPOSE 8080
ENTRYPOINT ["/app/app"]
