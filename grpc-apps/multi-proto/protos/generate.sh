#!/bin/bash

# Exit on error
set -e

echo "Generating Go code from proto files..."

# Generate Go code for svc/v1 proto files
# -I. includes the current directory (protos/)
# -I../third_party includes the third_party directory at multi-proto root
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    -I. \
    -I../third_party \
    svc/v1/*.proto

echo "✓ Proto files generated successfully!"
