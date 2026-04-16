# DNS API Server

A simple Go API server that provides DNS query endpoints for various record types.

## Features

- **A Records**: Query IPv4 addresses
- **AAAA Records**: Query IPv6 addresses
- **TXT Records**: Query text records
- **MX Records**: Query mail exchange records
- **SRV Records**: Query service records

## Running the Server

```bash
go run main.go
```

The server will start on port `8086`.

## API Endpoints

### A Record (IPv4)
```bash
curl "http://localhost:8086/dns/a?domain=google.com"
```

### AAAA Record (IPv6)
```bash
curl "http://localhost:8086/dns/aaaa?domain=google.com"
```

### TXT Record
```bash
curl "http://localhost:8086/dns/txt?domain=google.com"
```

### MX Record
```bash
curl "http://localhost:8086/dns/mx?domain=google.com"
```

### SRV Record
```bash
curl "http://localhost:8086/dns/srv?service=xmpp-server&proto=tcp&name=gmail.com"
```

### Health Check
```bash
curl "http://localhost:8086/health"
```

## Example Responses

### A Record Response
```json
{
  "domain": "google.com",
  "type": "A",
  "records": [
    "142.250.185.46"
  ]
}
```

### MX Record Response
```json
{
  "domain": "google.com",
  "type": "MX",
  "records": [
    {
      "host": "smtp.google.com.",
      "priority": 10
    }
  ]
}
```

### SRV Record Response
```json
{
  "domain": "_xmpp-server._tcp.gmail.com",
  "type": "SRV",
  "records": [
    {
      "target": "xmpp-server.l.google.com.",
      "port": 5269,
      "priority": 5,
      "weight": 0
    }
  ]
}
```
