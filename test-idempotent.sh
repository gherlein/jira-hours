#!/bin/bash

cd "$(dirname "$0")"

echo "=== Testing Idempotent Add ==="
echo

# Create a simple Go test program that uses the same MockClient twice
cat > /tmp/test-idempotent.go << 'EOF'
package main

import (
	"fmt"
	"time"
	"github.com/gherlein/jira-hours/internal/jira"
)

func main() {
	client := jira.NewMockClient()
	client.SetUserEmail("test@example.com")

	date := time.Date(2026, 2, 2, 8, 0, 0, 0, time.UTC)

	fmt.Println("First run - adding worklog:")
	err := client.AddWorklog("NPU-25", 1, date)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("✓ Worklog added")
	}

	fmt.Println("\nChecking if exists:")
	exists, id, err := client.WorklogExists("NPU-25", date, 1, "test@example.com")
	if err != nil {
		fmt.Printf("Error checking: %v\n", err)
	} else if exists {
		fmt.Printf("✓ Worklog exists (ID: %s)\n", id)
	} else {
		fmt.Println("✗ Worklog not found")
	}

	fmt.Println("\nSecond run - should detect existing:")
	exists, id, err = client.WorklogExists("NPU-25", date, 1, "test@example.com")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else if exists {
		fmt.Printf("○ Already logged (worklog %s) - would skip\n", id)
	} else {
		fmt.Println("Would add new worklog")
	}
}
EOF

cd /tmp
go mod init test-idempotent 2>&1 > /dev/null
cp -r /Users/gherlein/herlein/src/hours/go.mod .
cp -r /Users/gherlein/herlein/src/hours/go.sum . 2>/dev/null || true
go run test-idempotent.go

echo
echo "=== Test Complete ==="
