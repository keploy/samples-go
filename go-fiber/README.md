# Product Management API

A high-performance REST API built with Go Fiber, PostgreSQL, and Redis for managing products with advanced features like ratings, tags, analytics, and shopping carts.

## Features

- **Product Management**: CRUD operations for products with metadata support
- **Rating System**: Rate products and get rating summaries
- **Tagging System**: Add tags to products and search by tags
- **Shopping Cart**: Redis-based cart management
- **Analytics**: Activity logging, visitor tracking, and leaderboards
- **Performance**: PostgreSQL for persistence, Redis for caching and analytics

## Quick Start

### Using Docker Compose

1. Clone the repository
2. Run: `docker-compose up -d`
3. API will be available at http://localhost:8080

### Manual Setup

1. Install dependencies: `go mod tidy`
2. Set environment variables:
   ```bash
   export DATABASE_URL="postgres://user:pass@localhost/dbname?sslmode=disable"
   export REDIS_URL="redis://localhost:6379"
   export PORT="8080"
   ```
3. Run: `go run cmd/server/main.go`

## API Endpoints

### Products
- `GET /api/v1/products` - List all products
- `POST /api/v1/products` - Create a product
- `GET /api/v1/products/:id` - Get a product
- `PUT /api/v1/products/:id` - Update a product
- `DELETE /api/v1/products/:id` - Delete a product
- `POST /api/v1/products/bulk` - Bulk create products

### Ratings
- `POST /api/v1/products/:id/rate` - Rate a product
- `GET /api/v1/products/:id/ratings` - Get product ratings

### Tags
- `POST /api/v1/products/:id/tags` - Add tags to product
- `GET /api/v1/products/:id/tags` - Get product tags
- `GET /api/v1/tags/:tag/products` - Get products by tag

### Cart
- `POST /api/v1/carts/:userId` - Update user cart
- `GET /api/v1/carts/:userId` - Get user cart

### Analytics
- `GET /api/v1/activity` - Get activity log
- `GET /api/v1/products/:id/visitors` - Get visitor count
- `GET /api/v1/leaderboard` - Get sales leaderboard

### Health
- `GET /health` - Health check

## Architecture

- **Clean Architecture**: Separation of concerns with handlers, services, and repositories
- **PostgreSQL**: Primary database for persistent data
- **Redis**: Caching, sessions, analytics, and real-time features
- **Fiber**: High-performance HTTP framework
- **UUID**: For all entity identifiers

## Performance Features

- Connection pooling for both PostgreSQL and Redis
- Prepared statements for database queries
- Pipeline operations for Redis bulk operations
- Proper indexing on frequently queried columns
- Efficient JSON handling for metadata