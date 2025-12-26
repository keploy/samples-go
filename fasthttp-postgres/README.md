# FastHTTP PostgreSQL Quickstart

A sample application demonstrating how to build a FastHTTP server with PostgreSQL database integration.

## Prerequisites

### For Local Development (without Docker)
- Go 1.21 or higher
- PostgreSQL 10.5 or higher (running locally)

### For Docker Setup
- Docker and Docker Compose installed

---

## Running with Docker Compose

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/keploy/samples-go.git
cd samples-go
```

2. Navigate to the fasthttp-postgres directory:
```bash
cd fasthttp-postgres
```

3. Start the application and database:
```bash
docker-compose up -d
```

The application will be available at `http://localhost:8080`

4. Test the API: 
```bash
curl http://localhost:8080/books
curl http://localhost:8080/authors
```

### Run Keploy Tests with Docker Compose

```bash
keploy test -c "docker-compose up" --containerName fasthttp-app --delay 10
```

### Stop the Services

```bash
docker-compose down
```

### Clean Up (Remove Database Data)

```bash
docker-compose down -v
```

### Custom Configuration

Edit environment variables in `docker-compose.yml` under the `fasthttp-app` service if needed. 

---

## Running Without Docker (Local Development)

### 1. Install PostgreSQL

**macOS (using Homebrew):**
```bash
brew install postgresql@10
brew services start postgresql@10
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib
sudo systemctl start postgresql
```

**Windows:**
Download and install from [postgresql.org](https://www.postgresql.org/download/windows/)

### 2. Create Database and User

```bash
psql -U postgres
```

Then run in the PostgreSQL prompt: 

```sql
CREATE DATABASE db;
```

Exit the prompt with `\q`

### 3. Run Database Migrations

If you have migration files, run them:

```bash
psql -U postgres -d db -f migrations/init.sql
```

### 4. Set Environment Variables (Optional)

By default, the app connects to `localhost:5432` with user `postgres` and password `password` on database `db`.

If you need to use different credentials, set these environment variables:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=password
export DB_NAME=db
```

### 5. Install Go Dependencies

```bash
cd fasthttp-postgres
go mod download
```

### 6. Run the Application

```bash
go run main.go
```

The application will be available at `http://localhost:8080`

### 7. Test the API

In another terminal: 

```bash
curl http://localhost:8080/books
curl http://localhost:8080/authors
```

### 8. Run Keploy Tests (Local)

```bash
keploy test -c "go run main.go" --delay 10
```

### 9. Stop the Application

Press `Ctrl+C` in the terminal running the application. 

---

## Project Structure

```
fasthttp-postgres/
├── Dockerfile
├── docker-compose.yml
├── main.go
├── go.mod
├── go.sum
├── README.md
├── internal/
│   ├── handlers/
│   │   └── handler.go
│   └── repository/
│       └── repository.go
└── migrations/
    └── init.sql
```

---

## Troubleshooting

### Docker Compose Issues

**Warning about version attribute:**
This is informational.

**Container fails to start:**
```bash
docker-compose logs fasthttp-app
```

**Port already in use:**
- Check what's using port 8080: `lsof -i :8080` (macOS/Linux)
- Change the port in `docker-compose.yml`:
```yaml
ports:
  - "9090:8080"  # Access app at localhost:9090
```

**Database connection timeout:**
- Ensure PostgreSQL container is running: `docker-compose ps`
- Check environment variables match in `docker-compose.yml`

### Local Development Issues

**PostgreSQL connection refused:**
- Ensure PostgreSQL is running: 
  - macOS: `brew services start postgresql@10`
  - Linux: `sudo systemctl status postgresql`
  - Windows: Check Services app

**Port 8080 already in use:**
- Check what's using port 8080: `lsof -i :8080`
- Kill the process: `kill -9 <PID>`

**Database does not exist:**
- Create it: `psql -U postgres -c "CREATE DATABASE db;"`
- Run migrations: `psql -U postgres -d db -f migrations/init.sql`

**Module not found errors:**
- Install dependencies: `go mod download`
- Update dependencies: `go mod tidy`

---

## API Endpoints

The application provides the following endpoints:

- `GET /books` - Get all books
- `GET /books/{id}` - Get book by ID
- `POST /books` - Create a new book
- `GET /authors` - Get all authors
- `GET /authors/{id}` - Get books by author ID
- `POST /authors` - Create a new author

---

## Next Steps

- Explore the code in `internal/handlers/` and `internal/repository/`
- Add more API endpoints as needed
- Set up Keploy for API testing and mocking
- Deploy to your preferred container orchestration platform (Kubernetes, Docker Swarm, etc.)

---

## Contributing

Contributions are welcome! Please make sure to:
- Follow the existing project structure
- Test your changes with both Docker Compose and local methods
- Update documentation if needed
