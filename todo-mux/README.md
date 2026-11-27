# Todo-Go Application

This is a RESTful Todo application written in Go, using PostgreSQL as the database and Gorilla Mux for routing. The application includes JWT authentication, request logging, and idempotency handling. It is written for testing keploy idempotency feature.

## Prerequisites

- Go 1.16 or later
- Docker and Docker Compose
- PostgreSQL

## Getting Started

### 1. Start the PostgreSQL database

Start the PostgreSQL database using Docker Compose:

```sh
docker-compose up -d
```

To stop and remove the volume and its data, use:

```sh
docker-compose down -v
```

### 2. Run the application

Run the Go application:

```sh
go build .
./todo-go
```

The server will start running on `http://localhost:3040`.

## Authentication

This application uses JWT for authentication. Before accessing protected endpoints, you need to obtain a token:

```sh
curl -X POST -H "Content-Type: application/json" -d '{"username": "admin", "password": "password"}' http://localhost:3040/login
```

The response will contain a token that should be used in subsequent requests:

```json
{"token":"your_jwt_token"}
```

## API Endpoints

### Login (Public)

```sh
curl -X POST -H "Content-Type: application/json" -d '{"username": "admin", "password": "password"}' http://localhost:3040/login
```

### Create a To-Do

```sh
curl -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_jwt_token" \
  -H "Idempotency-Key: unique_key" \
  -d '{"task": "Learn Go", "progress": "Todo"}' \
  http://localhost:3040/api/todos
```

Note: The Idempotency-Key header is required to prevent duplicate creation of todos.

### Get All To-Dos

```sh
curl -H "Authorization: Bearer your_jwt_token" http://localhost:3040/api/todos
```

### Get a Specific To-Do

```sh
curl -H "Authorization: Bearer your_jwt_token" http://localhost:3040/api/todos/1
```

### Update a To-Do

```sh
curl -X PUT \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_jwt_token" \
  -d '{"task": "Learn Keploy", "progress": "Done"}' \
  http://localhost:3040/api/todos/1
```

### Delete a To-Do

```sh
curl -X DELETE -H "Authorization: Bearer your_jwt_token" http://localhost:3040/api/todos/1
```

## Features

- **JWT Authentication**: Secure API endpoints with JWT tokens
- **Request Logging**: Each request is logged with a unique request ID
- **Idempotency Keys**: Prevent duplicate creation of resources
- **Error Handling**: Proper error responses with appropriate HTTP status codes
- **RESTful API Design**: Follow REST principles for API design

## Response Format

Most endpoints return responses in the following format:

```json
{
  "request_id": "unique-request-id",
  "timestamp": "2025-03-12T12:00:00Z",
  "todo": {
    "id": 1,
    "task": "Learn Go",
    "progress": "Todo",
    "last_checked": "2025-03-12T12:00:00Z"
  }
}
```

## License

This project is licensed under the MIT License.
