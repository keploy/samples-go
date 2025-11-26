# Multi-Proto gRPC Application

A gRPC server application demonstrating multiple protocol buffer files with complex data types including strings, integers, booleans, nested objects, maps, repeated fields, and timestamp usage.

## Purpose

This sample app helps Keploy verify whether the Protoscope-to-JSON feature is functioning correctly.

## Prerequisites

Before you begin, ensure you have Go installed on your system.

## Generate Go Files from Proto Definitions

### 1. Install Required Tools

Install the Protocol Buffer compiler and Go plugins:

```bash
# Install protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Install protoc-gen-go-grpc
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure the protoc compiler is installed
# For macOS:
brew install protobuf

# For Linux (Ubuntu/Debian):
sudo apt update
sudo apt install -y protobuf-compiler
```

Make sure `$GOPATH/bin` is in your PATH:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### 2. Generate Go Code

Navigate to the protos directory and run the generation script:

```bash
cd protos
chmod +x generate.sh
./generate.sh
```

This will generate Go code for all proto files in the `svc/v1` directory.

## Install grpcurl

grpcurl is a command-line tool for interacting with gRPC servers.

### Installation

```bash
# Install grpcurl using Go
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### Set PATH

Ensure `grpcurl` is in your PATH:

```bash
# Add this to your ~/.bashrc, ~/.zshrc, or equivalent
export PATH="$PATH:$(go env GOPATH)/bin"

# Reload your shell configuration
source ~/.bashrc  # or source ~/.zshrc
```

Verify installation:

```bash
grpcurl --version
```

## Run the Application

### 1. Install Dependencies

Navigate to the server directory and install dependencies:

```bash
cd server
go mod tidy
```

### 2. Start the Server

```bash
go run main.go
```

The server will start listening on port `50051`.

## Test with grpcurl

### GetSimpleData API

```bash
grpcurl -plaintext -d '{"id": "123"}' localhost:50051 svc.v1.DataService/GetSimpleData
```

Expected response:
```json
{
  "message": "Simple data retrieved successfully",
  "code": 200,
  "success": true,
  "value": 123.45,
  "items": ["item1", "item2", "item3"],
  "metadata": {
    "key1": "value1",
    "key2": "value2"
  },
  "timestamp": "2025-11-26T..."
}
```

### GetComplexData API

```bash
grpcurl -plaintext -d '{"query": "test", "limit": 10}' localhost:50051 svc.v1.DataService/GetComplexData
```

Expected response includes users, products, statistics, and metadata with timestamps.

## List Available Services

To see all available services and methods:

```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 list svc.v1.DataService
```
