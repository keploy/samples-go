## Introduction

A sample Go API used to validate Keploy's debug packet capture (`.kpcap`)
across both plain HTTP and HTTPS traffic. Each useful endpoint also
talks to [PostgreSQL](https://www.postgresql.org/) and
[MongoDB](https://www.mongodb.com/) so dependency traffic is visible in
captures alongside the inbound HTTP/HTTPS request.

## Setup pcap-validator

```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/pcap-validator
go mod download
```

## Installation

```bash
curl --silent -O -L https://keploy.io/install.sh && source install.sh
```

## Running app using Docker

Keploy can be used on Linux, Windows and MacOS through [Docker](https://docs.docker.com/engine/install/).

> Note: To run Keploy on MacOS through [Docker](https://docs.docker.com/desktop/release-notes/#4252) the version must be `4.25.2` or above.

### Let's start the dependency containers

Create a shared network so the app and its deps can reach each other by name:

```bash
docker network create keploy-network
```

Start Postgres and Mongo on that network:

```bash
docker run -d --rm --name postgres --network keploy-network \
  -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=pcapdemo \
  -p 5432:5432 postgres:16-alpine

docker run -d --rm --name mongo --network keploy-network \
  -p 27017:27017 mongo:7
```

### Build the application image

```bash
docker build -t pcap-validator:1.0 .
```

### Capture the Testcases

```bash
keploy record -c "docker run -p 8080:8080 -p 8443:8443 --rm --name PcapApp --network keploy-network -e DATABASE_URL='postgres://postgres:postgres@postgres:5432/pcapdemo?sslmode=disable' -e MONGO_URI='mongodb://mongo:27017' -e MONGO_DATABASE='pcapdemo' pcap-validator:1.0"
```

To generate testcases just make some API calls. You can use
[Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/),
or simply `curl`.

#### 1) Health check

```bash
# plain
curl -s http://localhost:8080/healthz

# TLS (-k because the server creates an ephemeral self-signed cert)
curl -sk https://localhost:8443/healthz
```

#### 2) Create a user (writes to Postgres + Mongo)

```bash
# plain
curl -s -X POST http://localhost:8080/users \
  -H 'content-type: application/json' \
  -d '{"name":"Plain User","email":"plain@example.com"}'

# TLS
curl -sk -X POST https://localhost:8443/users \
  -H 'content-type: application/json' \
  -d '{"name":"TLS User","email":"tls@example.com"}'
```

#### 3) Touch (writes one row to Postgres and one document to Mongo)

```bash
# plain
curl -s -X POST http://localhost:8080/touch \
  -H 'content-type: application/json' \
  -d '{"label":"plain-http-touch"}'

# TLS
curl -sk -X POST https://localhost:8443/touch \
  -H 'content-type: application/json' \
  -d '{"label":"tls-https-touch"}'
```

#### 4) List users

```bash
curl -s http://localhost:8080/users
```

Both API calls are captured as testcases and surface in the Keploy CLI.
You'll see a `keploy/` folder with the test cases and mocks, and a
`keploy/debug/` folder with the `.kpcap` files when `--debug` is on.

### Run the captured testcases

```bash
keploy test -c "docker run -p 8080:8080 -p 8443:8443 --rm --name PcapApp --network keploy-network -e DATABASE_URL='postgres://postgres:postgres@postgres:5432/pcapdemo?sslmode=disable' -e MONGO_URI='mongodb://mongo:27017' -e MONGO_DATABASE='pcapdemo' pcap-validator:1.0" --delay 10
```

Replay reuses the recorded mocks for Postgres and Mongo, so the app
thinks it is talking to real databases without either dependency being
involved 😄.

## Run app natively on local machine

### Let's start the dependencies

Postgres and Mongo are brought up via the bundled `docker-compose.yml`:

```bash
docker compose up -d postgres mongo
```

### TLS variant

A second compose file, `docker-compose.tls.yml`, brings up the same
deps but with **Postgres `ssl=on`** and **Mongo `--tlsMode requireTLS`**.
A `certgen` init container generates an ephemeral self-signed cert at
compose-up time — no certs are committed to the repo.

```bash
docker compose -f docker-compose.tls.yml up -d certgen postgres mongo
```

Use these connection strings against the TLS variant:

```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/pcapdemo?sslmode=require'
MONGO_URI='mongodb://localhost:27017/?tls=true&tlsInsecure=true'
```

This is the path used by the encrypted-deps row of Keploy's
[`pcap_validator`](https://github.com/keploy/keploy/blob/main/.github/workflows/pcap_validator.yml)
workflow to verify that Keploy's Postgres / Mongo parsers correctly MITM
upstream TLS — the marker in the JSON body must still appear plaintext
in the outgoing `.kpcap` even though it travels TLS-encrypted on the
wire.

### Build the application

```bash
go build -o pcap-validator .
```

### Capture the Testcases

```bash
sudo -E env PATH=$PATH \
  DATABASE_URL='postgres://postgres:postgres@localhost:5432/pcapdemo?sslmode=disable' \
  MONGO_URI='mongodb://localhost:27017' \
  MONGO_DATABASE='pcapdemo' \
  keploy record -c "./pcap-validator" --debug
```

`--debug` enables Keploy's `.kpcap` packet-capture output under
`./keploy/debug/record-*.kpcap` (one file each for incoming and
outgoing flows). Use the same curl commands shown in the Docker
section above to drive traffic.

### Run the captured testcases

```bash
sudo -E env PATH=$PATH \
  DATABASE_URL='postgres://postgres:postgres@localhost:5432/pcapdemo?sslmode=disable' \
  MONGO_URI='mongodb://localhost:27017' \
  MONGO_DATABASE='pcapdemo' \
  keploy test -c "./pcap-validator" --delay 10
```

## API routes

- `GET /healthz` — checks Postgres and Mongo connectivity.
- `POST /users` — inserts a user into Postgres and an audit event into MongoDB.
- `GET /users` — lists recent users from Postgres and recent audit events from MongoDB.
- `POST /touch` — inserts one row into Postgres and one document into MongoDB.

## Capture targets at a glance

- Plain API traffic on port `8080`
- TLS API traffic on port `8443`
- Postgres dependency traffic on port `5432`
- Mongo dependency traffic on port `27017`

The API-to-Postgres and API-to-Mongo links in the default compose file
are intentionally unencrypted. The HTTPS listener is the encrypted path
for validating `.kpcap` handling of TLS traffic on the inbound side; the
TLS variant of the compose stack does the same for outbound dependency
traffic.
