#!/bin/bash

cd "$(dirname "$0")"

echo "=== Testing Jira Hours CLI ==="
echo

echo "1. Validating test data (2026-02)..."
./bin/jira-hours validate --month 2026-02
echo

echo "2. Running dry-run for test data..."
./bin/jira-hours log --month 2026-02 --dry-run
echo

echo "3. Running with mock client (no API calls)..."
./bin/jira-hours log --month 2026-02 --mock
echo

echo "=== Tests Complete ==="
