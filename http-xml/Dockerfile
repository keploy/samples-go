FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY . .

# Initialize go module if not present and build
RUN if [ ! -f go.mod ]; then go mod init http-xml; fi && \
    go build -o main .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 9090

CMD ["./main"]
