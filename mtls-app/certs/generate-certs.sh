#!/bin/sh
set -eu

CERT_DIR="${CERT_DIR:-/certs}"
HOST_UID="${HOST_UID:-}"
HOST_GID="${HOST_GID:-}"
mkdir -p "${CERT_DIR}"

cat > "${CERT_DIR}/server-openssl.cnf" <<'CNF'
[req]
distinguished_name = dn
req_extensions = req_ext
prompt = no

[dn]
CN = mtls-server

[req_ext]
subjectAltName = @alt_names
extendedKeyUsage = serverAuth

[alt_names]
DNS.1 = mtls-server
DNS.2 = localhost
CNF

cat > "${CERT_DIR}/client-openssl.cnf" <<'CNF'
[req]
distinguished_name = dn
req_extensions = req_ext
prompt = no

[dn]
CN = mtls-client

[req_ext]
extendedKeyUsage = clientAuth
CNF

openssl genrsa -out "${CERT_DIR}/ca.key" 2048
openssl req -x509 -new -nodes -key "${CERT_DIR}/ca.key" -sha256 -days 3650 \
  -out "${CERT_DIR}/ca.crt" -subj "/CN=mtls-demo-ca"

openssl genrsa -out "${CERT_DIR}/server.key" 2048
openssl req -new -key "${CERT_DIR}/server.key" -out "${CERT_DIR}/server.csr" \
  -config "${CERT_DIR}/server-openssl.cnf"
openssl x509 -req -in "${CERT_DIR}/server.csr" -CA "${CERT_DIR}/ca.crt" \
  -CAkey "${CERT_DIR}/ca.key" -CAcreateserial -out "${CERT_DIR}/server.crt" \
  -days 3650 -sha256 -extensions req_ext -extfile "${CERT_DIR}/server-openssl.cnf"

openssl genrsa -out "${CERT_DIR}/client.key" 2048
openssl req -new -key "${CERT_DIR}/client.key" -out "${CERT_DIR}/client.csr" \
  -config "${CERT_DIR}/client-openssl.cnf"
openssl x509 -req -in "${CERT_DIR}/client.csr" -CA "${CERT_DIR}/ca.crt" \
  -CAkey "${CERT_DIR}/ca.key" -CAcreateserial -out "${CERT_DIR}/client.crt" \
  -days 3650 -sha256 -extensions req_ext -extfile "${CERT_DIR}/client-openssl.cnf"

rm -f \
  "${CERT_DIR}/server.csr" \
  "${CERT_DIR}/client.csr" \
  "${CERT_DIR}/ca.srl" \
  "${CERT_DIR}/server-openssl.cnf" \
  "${CERT_DIR}/client-openssl.cnf"

chmod 600 \
  "${CERT_DIR}/ca.key" \
  "${CERT_DIR}/server.key" \
  "${CERT_DIR}/client.key"

chmod 644 \
  "${CERT_DIR}/ca.crt" \
  "${CERT_DIR}/server.crt" \
  "${CERT_DIR}/client.crt"

if [ -n "${HOST_UID}" ] && [ -n "${HOST_GID}" ]; then
  chown -R "${HOST_UID}:${HOST_GID}" "${CERT_DIR}"
fi

echo "certificates generated in ${CERT_DIR}"
