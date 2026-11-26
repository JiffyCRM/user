#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if ! command -v protoc >/dev/null 2>&1; then
  echo "protoc is required but was not found in PATH" >&2
  exit 1
fi

GOBIN="${GOBIN:-$(go env GOPATH)/bin}"
export PATH="$GOBIN:$PATH"

PROTOC_GEN_GO_VERSION="v1.36.6"
PROTOC_GEN_GO_GRPC_VERSION="v1.5.1"

go install "google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}"
go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}"

protoc \
  -I api/proto \
  --go_out=. \
  --go_opt=module=github.com/JiffyCRM/user \
  --go-grpc_out=. \
  --go-grpc_opt=module=github.com/JiffyCRM/user \
  api/proto/user/v1/user.proto

echo "generated api/user/v1"