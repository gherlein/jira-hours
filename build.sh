#!/bin/bash

cd "$(dirname "$0")"

echo "Building jira-hours CLI..."
go mod download
go build -o bin/jira-hours ./cmd/hours

if [ $? -eq 0 ]; then
    echo "✓ Build successful: bin/jira-hours"
else
    echo "✗ Build failed"
    exit 1
fi
