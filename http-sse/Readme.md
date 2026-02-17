
# HTTP + SSE Sample with Keploy

This sample demonstrates how to test a Go application that serves both standard **HTTP** endpoints and **Server-Sent Events (SSE)** on different ports using [Keploy](https://keploy.io).

It uses **Caddy** as a reverse proxy to route traffic to the appropriate ports based on the request path.

## Prerequisites

- [Go](https://go.dev/doc/install) (1.21 or later)
- [Docker](https://docs.docker.com/engine/install/) (for running Caddy)
- [Keploy](https://keploy.io/docs/server/installation/)

## Architecture

- **HTTP Server**: Listens on port `8000`. Handles API requests.
- **SSE Server**: Listens on port `8047`. Handles real-time event streams.
- **Caddy (Router)**: Listens on port `80`. Routes `/subscribe/*` to port `8047` and everything else to `8000`.

## Setup

### 1. Start the Caddy Router
Run the following command to start Caddy in a Docker container. This will simulate a production-like routing environment where a gateway handles traffic distribution.

```bash
docker rm -f sse-router 2>/dev/null || true
docker run --name sse-router --rm -p 80:80 \
  --add-host=host.docker.internal:host-gateway \
  -v "$PWD/Caddyfile.record:/etc/caddy/Caddyfile" \
  caddy:2-alpine
```

## Recording Test Cases

Open a new terminal to run the application with Keploy recording enabled.

### 1. Start the Application
```bash
keploy record
```

### 2. Generate Traffic
Send requests to the Caddy router (localhost:80).

**Standard HTTP Request:**
```bash
# 1. Health Check                                  
curl -i "http://localhost:8000/health"                                       

# 2. Simple Hello                                               
curl -i "http://localhost:8000/api/hello"               

# 3. Echo (with message)                                 
curl -i "http://localhost:8000/api/echo?msg=Greetings"                           

# 3b. Echo (empty/default message)
curl -i "http://localhost:8000/api/echo"

# 4. Add Numbers
curl -i "http://localhost:8000/api/add?a=50&b=25"

# 5. Get Current Time
curl -i "http://localhost:8000/api/time"

# 6. Get Resource by ID
curl -i "http://localhost:8000/api/resource/user-123"
```

**SSE Request (Student):**
```bash
curl -N -H "Accept: text/event-stream" \
  "http://localhost/subscribe/student/events?doubtId=abc"
```

**SSE Request (Teacher):**
```bash
curl -N -H "Accept: text/event-stream" \
  "http://localhost/subscribe/teacher/events?teacherId=t-42"
```

*Note: You can verify that traffic is being routed correctly by checking the "port" field in JSON responses or the server logs.*

## Running Tests

Once you have recorded the test cases, you can stop the recording (Ctrl+C) and run the tests.

```bash
keploy test
```

Keploy will replay the recorded traffic and verify that the responses match the expected output, ensuring that routing and logic remain consistent.