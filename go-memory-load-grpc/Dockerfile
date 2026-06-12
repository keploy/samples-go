FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN go build -o /bin/api ./cmd/api

FROM alpine:3.22

WORKDIR /app
COPY --from=build /bin/api /app/api

EXPOSE 8080
EXPOSE 50051

CMD ["/app/api"]
