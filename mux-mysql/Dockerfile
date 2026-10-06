FROM golang:1.21.4-alpine AS build
RUN mkdir /app
COPY . /app
WORKDIR /app

ENV ConnectionString="root:my-secret-pw@tcp(mysql:3306)/mysql"

# proxy.golang.org now and then resets a download mid-transfer ("stream error:
# ... INTERNAL_ERROR"), which fails an un-retried fetch and the whole build.
# Retried: 5 attempts, 5-20s apart, then fail.
RUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done
RUN go build -o main .
RUN chmod +x /app/main

#### Stage 2
FROM scratch
COPY --from=build /app/main /main
ENV ConnectionString="root:my-secret-pw@tcp(mysql:3306)/mysql"
ENTRYPOINT [ "/main" ]