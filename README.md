# Jira Time Logger

Automate logging hours from monthly time sheets to Jira tickets.

## Prerequisites

- Go 1.21 or later
- Jira Cloud account with API access
- Jira API token
- direnv (optional but recommended for .envrc)

## Installation

```bash
make all
```

The binary will be created at `bin/jira-hours`.

## Quick Start

```bash
# Show all available make targets
make

# Build the tool
make build

# Test with mock client (no API calls)
make run-test

# Validate your data
make validate-test
```

## Configuration

### Set up environment variables

Add to your `.envrc` or `.env` file:

```bash
export JIRA_CLOUD_ID="29ad0f88-9969-4673-b232-4aa64e95f11b"
export JIRA_EMAIL="your-email@brightsign.biz"
export JIRA_TOKEN="your-api-token-here"
```

Optional (defaults to https://api.atlassian.com):
```bash
export JIRA_BASE_URL="https://api.atlassian.com"
```

If using direnv:
```bash
direnv allow
```

### Get your Jira API token

1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token"
3. Give it a name like "jira-hours"
4. Copy the token and save it in your `.envrc` file

## Usage

### Validate time log

```bash
./bin/jira-hours validate --month 2026-01
```

### Dry run (see what would be logged)

```bash
./bin/jira-hours log --month 2026-01 --dry-run
```

### Log hours to Jira

```bash
./bin/jira-hours log --month 2026-01
```

### Log specific week only

```bash
./bin/jira-hours log --month 2026-01 --week 3
```

### Log specific project only

```bash
./bin/jira-hours log --month 2026-01 --code NPU
```

### Test with mock client (no API calls)

```bash
./bin/jira-hours log --month 2026-02 --mock
```

### Convert markdown to YAML

```bash
# Convert projects mapping
./bin/jira-hours convert --input old/PROJECTS.md --output configs/projects.yaml --projects

# Convert monthly time log
./bin/jira-hours convert --input old/202601.md --output data/2026-01.yaml --month 2026-01
```

## File Structure

```
hours/
├── bin/jira-hours          # Built binary
├── cmd/hours/              # CLI source code
├── internal/               # Library packages
├── configs/
│   └── projects.yaml       # Project code to Jira ticket mapping
├── data/
│   ├── 2026-01.yaml       # Monthly time logs
│   └── 2026-02.yaml       # Test data
└── old/                    # Archived markdown files
```

## Data Format

### Monthly Time Log (data/YYYY-MM.yaml)

```yaml
month: 2026-01
hours:
  - code: NPU
    week1: 12
    week2: 13
    week3: 10
    week4: 12
  - code: MGNT
    week1: 24
    week2: 10
    week3: 11
    week4: 13
```

### Project Config (configs/projects.yaml)

```yaml
projects:
  NPU:
    ticket: NPU-25
    description: Master Epic for Tracking AI/NPU work
  MGNT:
    ticket: BCN-13815
    description: Management Cloud
```

## How It Works

1. Reads monthly time log from `data/YYYY-MM.yaml`
2. Maps CAP codes to Jira ticket IDs using `configs/projects.yaml`
3. Calculates the Monday of each week (Week 1 = first Monday of month)
4. Calls Jira API to add worklog entries with the specified hours
5. Skips entries with 0 hours
6. Reports success/failure for each entry

## Date Calculation

The tool finds the first Monday of the month, then:
- Week 1 = First Monday
- Week 2 = First Monday + 7 days
- Week 3 = First Monday + 14 days
- Week 4 = First Monday + 21 days

Example for January 2026:
- January 1, 2026 is Thursday
- Week 1: Monday, January 5, 2026
- Week 2: Monday, January 12, 2026
- Week 3: Monday, January 19, 2026
- Week 4: Monday, January 26, 2026

## Testing

Test data file is at `data/2026-02.yaml` with 1 hour each for NPU (week 1) and MGNT (week 3).

Run tests:

```bash
make run-test
```

Or use the test script:

```bash
./test.sh
```

## Authentication

The tool requires environment variables for Jira authentication:

- `JIRA_CLOUD_ID` - Your Jira Cloud instance ID
- `JIRA_EMAIL` - Your Atlassian account email
- `JIRA_TOKEN` - Your Jira API token (not password)
- `JIRA_BASE_URL` - API base URL (optional, defaults to https://api.atlassian.com)

Set these in `.envrc` (with direnv) or `.env` file.

The tool uses Jira Cloud REST API with Basic Authentication:
- Combines email and API token as `email:token`
- Base64 encodes the string
- Sends in `Authorization: Basic <encoded>` header

This is the standard Jira Cloud authentication method.

## Troubleshooting

### "JIRA_CLOUD_ID environment variable required"
Set all required environment variables in `.envrc` or `.env`

### "connection test failed"
- Verify API token is correct
- Check cloud_id matches your Jira instance
- Ensure email has access to the Jira instance

### "unknown project code"
- Add the code to `configs/projects.yaml`
- Run validation: `./bin/jira-hours validate --month 2026-01`

### "week N extends beyond month"
- Some months don't have 4 full Mondays
- Adjust week assignments in time log

## Make Targets

Run `make` to see all available targets:

- `make build` - Build the binary
- `make test` - Run Go tests
- `make run-test` - Test with mock client
- `make validate-test` - Validate test data
- `make dry-run` - Show what would be logged
- `make clean` - Remove build artifacts
- `make all` - Download deps and build

See `make help` for complete list.
