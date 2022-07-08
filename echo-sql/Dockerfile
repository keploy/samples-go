FROM --platform=${BUILDPLATFORM} golang:alpine as go-builder

RUN apk add -U --no-cache ca-certificates && apk add build-base

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o echo-sql main.go handler.go

EXPOSE 8080

ENTRYPOINT ["/app/echo-sql"]