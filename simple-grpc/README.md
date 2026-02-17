# Simple gRPC Server

This is a sample gRPC server implementation in Go, demonstrating basic gRPC service methods.

## Overview

The `simple-grpc` application starts a gRPC server on port `50051`. It provides a **DemoService** with several RPC methods to showcase different interaction patterns.

## Features

The server implements the following methods:

1.  **Health**: Returns a simple boolean status.
2.  **Hello**: Responds with a greeting message.
3.  **Echo**: Returns the same message it received.
4.  **Add**: Takes two integers and returns their sum.
5.  **GetTime**: Returns the current server time in UTC.
6.  **GetResource**: Simulates fetching a resource by ID.

## Prerequisites

-   Go 1.23 or higher
-   Protobuf compiler (`protoc`) installed (if you want to regenerate files)

## Installation

1.  Clone the repository.
2.  Navigate to the `simple-grpc` directory:

    ```bash
    cd simple-grpc
    ```

3.  Download dependencies:

    ```bash
    go mod tidy
    ```

## Usage

### Running the Server

Start the server using `go run`:

```bash
go run main.go
```

You should see:

```text
2024/02/17 14:18:41 gRPC server listening on [::]:50051
```

### Testing with `grpcurl`

You can use `grpcurl` to interact with the server. Ensure `grpcurl` is installed.

#### 1. Health Check
```bash
grpcurl -plaintext -proto pb/demo.proto localhost:50051 demo.DemoService/Health
```

#### 2. Simple Hello
```bash
grpcurl -plaintext -proto pb/demo.proto localhost:50051 demo.DemoService/Hello
```

#### 3. Echo (with message)
```bash
grpcurl -plaintext -proto pb/demo.proto -d '{"msg": "Greetings"}' localhost:50051 demo.DemoService/Echo
```

#### 3b. Echo (empty/default message)
```bash
grpcurl -plaintext -proto pb/demo.proto -d '{}' localhost:50051 demo.DemoService/Echo
```

#### 4. Add Numbers (a=50, b=25)
```bash
grpcurl -plaintext -proto pb/demo.proto -d '{"a": 50, "b": 25}' localhost:50051 demo.DemoService/Add
```

#### 6. Get Resource by ID
```bash
grpcurl -plaintext -proto pb/demo.proto -d '{"id": "user-123"}' localhost:50051 demo.DemoService/GetResource
```

## Project Structure

-   `main.go`: The main entry point and server implementation.
-   `pb/`: Contains the generated gRPC code and `.proto` definition.
-   `go.mod` / `go.sum`: Go module dependency files.
