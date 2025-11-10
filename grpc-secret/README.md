## Introduction

This sample exposes a gRPC server that returns structured payloads containing synthetic secrets. The steps below show how to use Keploy on Linux to record traffic and replay it as tests.

## Setup

```bash
git clone https://github.com/keploy/samples-go.git
cd samples-go/grpc-secret
go mod download
```

## Prerequisites

- Go 1.21+ installed and in PATH
- Keploy CLI available in PATH as `keploy`
- grpcurl installed:

```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
export PATH="$PATH:$HOME/go/bin"
```

## Record → Hit 4 Endpoints → Replay (Linux)

1) Generate Keploy config (once):

```bash
keploy config --generate
```

2) Start recording in terminal A:

```bash
keploy record -c "go run ."
```

3) In terminal B, send the 4 gRPC calls (exact commands below):

```bash
grpcurl -plaintext -v \
  -H "authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.TCYt5XsITJX1CxPCT8yAV-TVkIEq_PbChOMqsLfRoPsnsgw5WEuts01mq-pQy7UJiN5mgRxD-WUcX16dUEMGlv50aTquLy4H2jJL6QRTQFXBF9kGvUKL9Q" \
  -H "x-api-key: AIzaSyD-9tNxCeDNj3S0K_abcdefghijklmnopqr" \
  -H "x-session-token: SG.abc123xyz.789def012ghi345jkl678mno901pqr234stu" \
  -H "x-mongodb-password: mongodb+srv://user:SecretP@ssw0rd123@cluster.net" \
  -H "x-postgres-url: postgres://dbuser:MyS3cr3tP@ss@db.internal:5432/prod" \
  -H "x-datadog-key: 0123456789abcdef0123456789abcdef" \
  -d '{}' \
  localhost:50051 \
  secrets.SecretService/JwtLab

grpcurl -plaintext -v \
  -H "authorization: Bearer wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" \
  -H "x-api-key: sk-1234567890abcdefghijklmnopqrstuvwxyzABCDEF" \
  -H "x-session-token: npm_abcdefghijklmnopqrstuvwxyz1234567890" \
  -H "x-github-token: ghp_9876543210ZYXWVUTSRQPONMLKJIHGFEDCBAzy" \
  -H "x-slack-webhook: https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX" \
  -H "x-twilio-token: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6" \
  -d '{}' \
  localhost:50051 \
  secrets.SecretService/CurlMix

grpcurl -plaintext -v \
  -H "authorization: Bearer sk_live_4eC39HqLyjWDarjtT1zdp7dc" \
  -H "x-api-key: AKIAJ5XXXXXXXXXXXXXXXX" \
  -H "x-session-token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh1234" \
  -H "x-hmac-secret: 1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z" \
  -H "x-cloudflare-token: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0" \
  -H "x-azure-key: DefaultEndpointsProtocol=https;AccountName=myaccount;AccountKey=abc123xyz789==" \
  -H "x-sendgrid-key: SG.abcdefghijklmnopqrstuv.xyz0123456789abcdefghijklmnopqrstuvwxyz" \
  -d '{}' \
  localhost:50051 \
  secrets.SecretService/Cdn

grpcurl -plaintext -v \
  -H "authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJrZXBsb3ktdGVzdHMiLCJzdWIiOiJ1c2VyXzMxMDBfYWJjZDEyMzQiLCJ0cyI6IjIwMjUtMDEtMDFUMDA6MDA6MDBaIn0.c29tZV9zaWduYXR1cmVfaGVyZQ" \
  -H "x-api-key: sk-proj-abcd1234efgh5678ijkl9012mnop3456qrst" \
  -H "x-session-token: ghp_1234567890abcdefghijklmnopqrstuvwxyz" \
  -H "x-stripe-key: sk_live_51aBc123dEfghIjkLmnOp456qrSTuvWxYz789" \
  -H "x-aws-access: AKIAIOSFODNN7EXAMPLE" \
  -H "x-mongodb-uri: mongodb+srv://admin:p4ssw0rd@cluster0.mongodb.net/mydb" \
  -d '{"id": 1}' \
  localhost:50051 \
  secrets.SecretService/GetSecret
```

4) Stop the recorder (Ctrl+C in terminal A).

5) Replay (run tests):

```bash
keploy test -c "go run ."
```

Reports are generated under `./keploy/reports/`.


