# Simple URL Shortening Service

This is a simple URL shortening application built with Node.js and MySQL, designed to test keploy.

## Requirements

- Docker & Docker Compose
- Node.js

## Setup

1. **Clone the Repository**:
git clone https://github.com/keploy/samples-go.git


2. **Navigate to the Project Directory**:
cd my-sql-js


3. **Start the MySQL Service using Docker**:
docker-compose up -d


4. **Install Dependencies and Start the Server**:
npm install
node app.js


## API Usage

### Create a Shortened URL

```bash
curl -X POST -H "Content-Type: application/json" -d '{"url": "https://example.com"}' http://localhost:3000/url
```

### Get an Existing Shortened URL

```bash
curl http://localhost:3000/[SHORTENED_ID]
```
### Update an Existing Shortened URL

```bash
curl -X PUT -H "Content-Type: application/json" -d '{"url": "https://newexample.com"}' http://localhost:3000/[SHORTENED_ID]
```

### Delete a Shortened URL
```bash
curl -X DELETE http://localhost:3000/[SHORTENED_ID]
```