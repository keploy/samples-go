# mTLS Go Sample

This sample demonstrates mutual TLS between a Go HTTPS server and a Go client. The server only accepts requests from clients that present a certificate signed by the shared demo CA, and the client verifies the server certificate before sending the request.

## Run with Docker Compose

```bash
docker compose up --build
```

What happens:

1. `cert-generator` creates a demo CA plus server and client certificates in a shared Docker volume.
2. `mtls-server` starts on `https://localhost:8443` and requires a valid client certificate.
3. `mtls-client` starts an API on `http://localhost:8080`.
4. Hitting `GET /hello` on the client API makes an mTLS request to `mtls-server` and returns the upstream response.

Try it:

```bash
curl http://localhost:8080/hello
```

## Run locally without Compose

Generate demo certificates:

```bash
docker build -t mtls-certs -f Dockerfile.certs .
docker run --rm \
  -e HOST_UID="$(id -u)" \
  -e HOST_GID="$(id -g)" \
  -v "$(pwd)/certs-local:/certs" \
  mtls-certs
```

Start the server:

```bash
SERVER_CERT_FILE="$(pwd)/certs-local/server.crt" \
SERVER_KEY_FILE="$(pwd)/certs-local/server.key" \
CA_CERT_FILE="$(pwd)/certs-local/ca.crt" \
go run ./cmd/server
```

In another terminal, run the client in API mode:

```bash
CLIENT_CERT_FILE="$(pwd)/certs-local/client.crt" \
CLIENT_KEY_FILE="$(pwd)/certs-local/client.key" \
CA_CERT_FILE="$(pwd)/certs-local/ca.crt" \
CLIENT_API_ADDR=":8080" \
SERVER_URL="https://localhost:8443/hello" \
go run ./cmd/client
```

Call the client API:

```bash
curl http://localhost:8080/hello
```

Optional: one-shot client mode (no API server) is still available by omitting `CLIENT_API_ADDR`.

## Big Payload Mode (50KB to 3MB)

Run the client in big payload mode:

```bash
CLIENT_CERT_FILE="$(pwd)/certs-local/client.crt" \
CLIENT_KEY_FILE="$(pwd)/certs-local/client.key" \
CA_CERT_FILE="$(pwd)/certs-local/ca.crt" \
CLIENT_API_ADDR=":8080" \
CLIENT_MODE="bigpayload" \
BIGPAYLOAD_SERVER_URL="https://localhost:8443/payload" \
go run ./cmd/client
```

The client exposes:

- `POST /bigpayload` for large payload testing
- request and response sizes must be between `51200` bytes (50KB) and `3145728` bytes (3MB)
- response size defaults to request size, but can be overridden with header `X-Response-Size-Bytes` or query `response_size_bytes`

Examples:

```bash
# 50KB request, 50KB response
head -c 51200 /dev/zero | curl -sS \
  -X POST http://localhost:8080/bigpayload \
  -H "Content-Type: application/octet-stream" \
  --data-binary @- \
  -o /tmp/resp-50kb.bin
wc -c /tmp/resp-50kb.bin
```

```bash
# 1MB request, 2MB response
head -c 1048576 /dev/zero | curl -sS \
  -X POST "http://localhost:8080/bigpayload?response_size_bytes=2097152" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @- \
  -o /tmp/resp-2mb.bin
wc -c /tmp/resp-2mb.bin
```

```bash
# 3MB request, 3MB response
head -c 3145728 /dev/zero | curl -sS \
  -X POST http://localhost:8080/bigpayload \
  -H "Content-Type: application/octet-stream" \
  --data-binary @- \
  -o /tmp/resp-3mb.bin
wc -c /tmp/resp-3mb.bin
```

The client logs upstream payload sizes after every request, and the server logs the handled request/response sizes too.
