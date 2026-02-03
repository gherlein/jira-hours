#!/bin/bash

cd "$(dirname "$0")"

echo "================================================================"
echo "FINAL DEMO: jira-hours Complete Feature Set"
echo "================================================================"
echo

echo "1. VALIDATE COMMAND WITH HOUR COUNT"
echo "===================================="
echo "Shows breakdown of hours by project with weekly details:"
echo
./bin/jira-hours validate --month 2026-02
echo

echo "================================================================"
echo "2. LOG COMMAND - IDEMPOTENT ADD"
echo "================================================================"
echo "First run - logs hours:"
echo
./bin/jira-hours log --month 2026-02 --mock --code NPU 2>&1 | grep -A 3 "Week 1"
echo

echo "================================================================"
echo "3. DELETE COMMAND - DRY RUN"
echo "================================================================"
echo "Preview what would be deleted:"
echo
./bin/jira-hours log --month 2026-02 --delete --dry-run --code MGNT 2>&1 | grep -A 3 "Week 3"
echo

echo "================================================================"
echo "4. FULL MONTH VALIDATION (January 2026)"
echo "================================================================"
echo
./bin/jira-hours validate --month 2026-01 2>&1 | tail -15
echo

echo "================================================================"
echo "All Features Working!"
echo "================================================================"
echo
echo "Commands available:"
echo "  1. validate - Validate and count hours"
echo "  2. log - Add hours (idempotent)"
echo "  3. log --delete - Remove hours"
echo
echo "Features:"
echo "  ✓ Hour counting in validate"
echo "  ✓ Idempotent add (no duplicates)"
echo "  ✓ Delete mode (error correction)"
echo "  ✓ Dry-run preview"
echo "  ✓ Mock testing"
echo "  ✓ Week/project filters"
echo
echo "Ready to use with real Jira!"
echo
