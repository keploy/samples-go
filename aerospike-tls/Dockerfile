FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /out/aerospike-tls-sample .

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/aerospike-tls-sample /aerospike-tls-sample
EXPOSE 8080
ENTRYPOINT ["/aerospike-tls-sample"]
