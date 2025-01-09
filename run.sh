#!/bin/bash
set -e

# Navigate to the qart directory
cd "$(dirname "$0")/qart"

echo "Building WebAssembly file..."
GOOS=js GOARCH=wasm go build -o main.wasm wasm.go arrow.go qr.go

echo "Starting local server..."
python3 -m http.server
