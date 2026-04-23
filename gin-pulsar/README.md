# Gin + Apache Pulsar Sample Application

A sample Go application demonstrating [Keploy](https://keploy.io) integration with [Apache Pulsar](https://pulsar.apache.org/) using the [Gin](https://gin-gonic.com/) web framework.

## Prerequisites

- [Docker](https://www.docker.com/) & Docker Compose
- [Go](https://golang.org/) 1.23+

## Quick Start

### Using Docker Compose

```bash
docker compose up --build
```

The app runs on `http://localhost:8090` with Pulsar on `localhost:6650`.

### Running Locally

Start Pulsar:

```bash
docker run -d --name pulsar -p 6650:6650 -p 8080:8080 apachepulsar/pulsar:4.0.3 bin/pulsar standalone
```

Run the app:

```bash
go run main.go
```

## API Endpoints

### Health Check
```bash
curl http://localhost:8090/healthz
```

### Publish a Message
```bash
curl -X POST http://localhost:8090/publish \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello Pulsar!", "key": "my-key"}'
```

### Get Received Messages
```bash
curl http://localhost:8090/messages
```

## Using with Keploy

```bash
keploy record -c "docker compose up --build" --container-name gin-pulsar-app
```

Then make API calls to generate test cases. Replay with:

```bash
keploy test -c "docker compose up --build" --container-name gin-pulsar-app
```
