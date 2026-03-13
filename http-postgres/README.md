# http-postgres

`http-postgres` is a small Go HTTP API backed by PostgreSQL. It is designed as a Keploy sample application for recording and replaying HTTP traffic together with Postgres interactions.

The app starts with `net/http`, connects to Postgres using `lib/pq`, runs startup migrations automatically, and exposes two resources:
- `companies` with a serial integer ID
- `projects` with a UUID ID

## Endpoints

### Companies
- `POST /companies`
- `GET /companies`
- `GET /companies/{name}`

### Projects
- `POST /projects`
- `GET /projects`
- `GET /projects/{id}`
- `PUT /projects/{id}`

## Prerequisites

- Go `1.24+`
- Docker and Docker Compose
- Keploy CLI

Install Keploy:

```bash
curl --silent -O -L https://keploy.io/install.sh
source install.sh
```

## Setup

```bash
git clone https://github.com/keploy/samples-go.git
cd samples-go/http-postgres
go mod download
```

## Run with Docker

Start the application and Postgres:

```bash
docker compose up --build
```

The API will be available at `http://localhost:8080`.

## Record Testcases with Keploy

Start recording:

```bash
keploy record -c "docker compose up" --container-name api --delay 10
```

In another terminal, generate traffic using the bundled scripts:

```bash
./test.sh
./test_projects.sh
```

This records API testcases and Postgres mocks into the `keploy/` directory.

## Replay Recorded Tests

Run the recorded testcases:

```bash
keploy test -c "docker compose up" --container-name api --delay 20
```

Keploy will replay the captured HTTP calls and mock the Postgres interactions during the test run.

## Run Natively

If you want to run the app without Docker for the API process, start only Postgres through Compose:

```bash
docker compose up -d db
```

Then run the server with local environment variables:

```bash
DB_HOST=localhost \
DB_PORT=5433 \
DB_USER=postgres \
DB_PASSWORD=postgres \
DB_NAME=pg_replicate \
go run .
```

## Notes

- Database tables are created automatically on startup.
- `test.sh` focuses on company creation, duplicates, validation, and listing flows.
- `test_projects.sh` exercises UUID-based project create, get, update, and list flows.
