# HTTP XML Sample

This is a simple Go application that serves an XML response.

## Build Docker Image

```bash
docker build -t http-xml-app .
```

## Run Docker Container

```bash
docker run -p 9090:9090 http-xml-app
```

## Test with Curl

You can test the application using the following curl command:

```bash
curl -v http://localhost:9090/xml
```
