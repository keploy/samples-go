# gRPC & HTTP Server Setup with Keploy

This guide explains how to set up and run the gRPC and HTTP servers using Keploy to capture and test the HTTP calls.
Here support grpc client (Unary rpc) has been showed, where keploy is recording incoming http calls and outgoing grpc calls. 

## Steps to Run

### 1. Start the gRPC Server

Open a terminal and navigate to the gRPC server directory:

```bash
cd grpc/grpc-server
```

Run the gRPC server:

```bash
go run server.go
```

### 2. Start the HTTP Server in Record Mode

Open another terminal and navigate to the HTTP server directory:

```bash
cd http
```

Build the HTTP server:

```bash
go build -o httpserver
```

Run the HTTP server in record mode to capture the incoming HTTP calls and outgoing gRPC client calls:

```bash
keploy record -c "./httpserver"
```

Make a curl request to the http server :
```bash
curl "http://localhost:8080/hello?name=Jack"
```

### 3. Run the HTTP Server in Test Mode

After recording, run the HTTP server in test mode to replay the captured calls:

```bash
keploy test -c "./httpserver"
```

---
