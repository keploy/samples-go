FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o echo-mysql .

FROM alpine:3.17
WORKDIR /app
COPY .env .
COPY --from=build /src/echo-mysql .
EXPOSE 9090
CMD ["./echo-mysql"]
