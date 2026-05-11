#!/usr/bin/env bash
# Generate a self-signed PKI for the TLS sample:
#   certs/ca.{pem,key}             — root CA
#   certs/server.{pem,key}         — Aerospike server cert with
#                                    CN/SAN = aerospike.local (and 127.0.0.1)
#   certs/client.{pem,key}         — client cert (optional mTLS)
#
# Re-run any time. Existing files are overwritten.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)/certs"
mkdir -p "$DIR"
cd "$DIR"

TLS_NAME="${TLS_NAME:-aerospike.local}"
DAYS="${DAYS:-825}"

cat > openssl.cnf <<EOF
[req]
distinguished_name = dn
prompt             = no
[dn]
CN = ${TLS_NAME}
[v3_server]
subjectAltName     = DNS:${TLS_NAME}, DNS:localhost, IP:127.0.0.1
extendedKeyUsage   = serverAuth
keyUsage           = digitalSignature, keyEncipherment
[v3_client]
subjectAltName     = DNS:aerospike-client
extendedKeyUsage   = clientAuth
keyUsage           = digitalSignature, keyEncipherment
EOF

# Root CA
openssl req -x509 -newkey rsa:2048 -nodes -days "$DAYS" \
  -keyout ca.key -out ca.pem \
  -subj "/CN=aerospike-tls-sample-CA"

# Server CSR + cert
openssl req -newkey rsa:2048 -nodes \
  -keyout server.key -out server.csr \
  -config openssl.cnf
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca.key -CAcreateserial \
  -out server.pem -days "$DAYS" \
  -extfile openssl.cnf -extensions v3_server

# Client CSR + cert
openssl req -newkey rsa:2048 -nodes \
  -keyout client.key -out client.csr \
  -subj "/CN=aerospike-client"
openssl x509 -req -in client.csr -CA ca.pem -CAkey ca.key -CAcreateserial \
  -out client.pem -days "$DAYS" \
  -extfile openssl.cnf -extensions v3_client

rm -f server.csr client.csr openssl.cnf ca.srl
chmod 644 ./*.pem
chmod 600 ./*.key

echo "Wrote certs to $DIR:"
ls -la "$DIR"
