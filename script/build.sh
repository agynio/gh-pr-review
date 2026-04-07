#!/bin/bash
set -e

mkdir -p dist

# Build for Windows
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/windows-amd64.exe .

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/darwin-amd64 .
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/darwin-arm64 .

# Build for Linux
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/linux-amd64 .
GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/linux-arm64 .
