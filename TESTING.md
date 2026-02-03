# Testing Instructions

## Quick Test

From the hours directory:

```bash
cd /Users/gherlein/herlein/src/hours

# Show all available make targets
make

# Build the tool
make build

# Run validation test
make validate-test

# Run mock test (no API calls)
make run-test

# Or run all tests
make run
```

## Makefile Targets

When you run `make` with no arguments, it displays all available targets:

- **help** - Display available targets (default)
- **all** - Install dependencies and build
- **deps** - Download Go dependencies
- **build** - Build the jira-hours binary
- **install** - Install to GOPATH/bin
- **test** - Run Go unit tests
- **fmt** - Format Go code
- **vet** - Run go vet
- **check** - Run formatting, vetting, and tests
- **clean** - Remove build artifacts
- **validate-test** - Validate test data
- **run-test** - Run with mock client
- **dry-run** - Show what would be logged for Jan 2026
- **run** - Run full test suite
- **convert-projects** - Convert PROJECTS.md to YAML
- **convert-month** - Convert 202601.md to YAML
- **build-all** - Build for all platforms

## Test Results

The mock test successfully:
- ✓ Logs 1 hour to NPU-25 on Monday, Feb 2, 2026
- ✓ Logs 1 hour to BCN-13815 on Monday, Feb 16, 2026
- ✓ Skips 46 zero-hour entries
- ✓ Totals 2 hours

## Authentication for Real Usage

To use with real Jira API, set environment variables in `.envrc`:

```bash
export JIRA_CLOUD_ID="29ad0f88-9969-4673-b232-4aa64e95f11b"
export JIRA_EMAIL="gherlein@brightsign.biz"
export JIRA_TOKEN="<your-token-here>"
```

Then run:
```bash
direnv allow
make dry-run  # Test first
./bin/jira-hours log --month 2026-01  # Actually log
```

**Note**: Only environment variables are supported (.envrc or .env). No config files.

## Files Created

```
hours/
├── bin/jira-hours              ← Built binary
├── cmd/hours/                  ← CLI source
├── internal/                   ← Library packages
├── configs/
│   └── projects.yaml          ← CAP code mappings
├── data/
│   ├── 2026-01.yaml          ← January hours
│   └── 2026-02.yaml          ← Test data
├── old/                       ← Archived markdown
├── Makefile                   ← Build automation
└── README.md                  ← Full documentation
```

## Ready to Use

The tool is complete and tested. When ready for production:

1. Set up environment variables in `.envrc`
2. Run: `direnv allow`
3. Run: `make validate-test`
4. Run: `make dry-run`
5. Run: `./bin/jira-hours log --month 2026-01`

Authentication uses your existing `.envrc` setup - no additional files needed!
