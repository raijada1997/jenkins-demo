#!/bin/bash
set -e

echo "🔨 Building Go service"

go version
go mod tidy
go build -o app

echo "✅ Build completed"

