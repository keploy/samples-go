# Simple URL Shortening Service

This is a simple URL shortening application built with Go and MySQL, designed to test keploy.

## Requirements

- Docker & Docker Compose
- Golang

## Setup

1. **Clone the Repository**:
git clone https://github.com/keploy/samples-go.git


2. **Navigate to the Project Directory**:
cd my-sql-go


3. **Start the MySQL Service using Docker**:
docker-compose up -d


4. **Install Dependencies and Start the Server**:
go mod tidy
go run main.go handler


## API Usage

### Create a Shortened URL

```bash
curl -X POST -H "Content-Type: application/json" -d '{"url":"https://www.example.com"}' http://localhost:8082/url
```

### Get an Existing Shortened URL

```bash
curl -X GET http://localhost:8082/url/{id}
```
### Update an Existing Shortened URL

```bash
curl -X PUT -H "Content-Type: application/json" -d '{"url":"https://www.newurl.com"}' http://localhost:8082/url/{id}
```

### Delete a Shortened URL
```bash
curl -X DELETE http://localhost:8082/url/{id}
```