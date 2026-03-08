#!/bin/bash

# Build Go binaries
echo "Building Go binaries..."
go build -o bin/mc cmd/mc/main.go
go build -o bin/ask cmd/ask/main.go
go build -o bin/askc cmd/askc/main.go

# Make binaries executable
chmod +x bin/mc
chmod +x bin/ask
chmod +x bin/askc

echo "Build complete. Binaries are in ./bin/"
echo "Run ./update_aliases.sh to install aliases"
