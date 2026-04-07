#!/bin/bash
set -e

# Build for current platform
go build -o gh-pr-review .

# Build for Windows (produces .exe)
GOOS=windows GOARCH=amd64 go build -o gh-pr-review-windows-amd64.exe .
zip gh-pr-review-windows-amd64.zip gh-pr-review-windows-amd64.exe

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o gh-pr-review-darwin-amd64 .
GOOS=darwin GOARCH=arm64 go build -o gh-pr-review-darwin-arm64 .
zip gh-pr-review-darwin-amd64.zip gh-pr-review-darwin-amd64
zip gh-pr-review-darwin-arm64.zip gh-pr-review-darwin-arm64

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o gh-pr-review-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o gh-pr-review-linux-arm64 .
zip gh-pr-review-linux-amd64.zip gh-pr-review-linux-amd64
zip gh-pr-review-linux-arm64.zip gh-pr-review-linux-arm64
