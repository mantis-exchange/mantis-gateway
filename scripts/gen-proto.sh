#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")/.."

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0

make proto

echo "Proto generation complete."
