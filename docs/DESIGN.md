# Jira Time Logger Design

## Overview

A Go CLI tool that reads monthly time logs, maps capitalization codes to Jira tickets, and automatically logs work hours via the Jira REST API.

## Goals

1. Automate logging hours from monthly time sheets to Jira
2. Support bulk updates for an entire month
3. Handle date calculations for week-based entries
4. Provide clear error messages and validation
5. Enable error correction via delete mode
6. Prevent duplicate logging with idempotent behavior

## Key Features

### Idempotent Logging

Running the log command multiple times with the same data won't create duplicates. Before adding a worklog:
- Checks if identical worklog exists (same date, same hours, same user)
- Skips if already present
- Reports as "already logged" with worklog ID

### Error Correction via Delete Mode

If hours are logged incorrectly:
- Use `--delete` flag to remove existing worklogs
- Finds worklogs by date and author
- Deletes only worklogs created by authenticated user
- Safe to re-run (deletes what matches, skips what doesn't exist)

### Workflow for Corrections

1. Discover error in logged hours
2. Fix data file (data/YYYY-MM.yaml)
3. Delete incorrect entries: `jira-hours log --month YYYY-MM --delete --dry-run` (verify)
4. Delete incorrect entries: `jira-hours log --month YYYY-MM --delete` (execute)
5. Re-log correct hours: `jira-hours log --month YYYY-MM`

## Architecture

### Components

```
┌─────────────────┐
│   CLI Entry     │
│   (main.go)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Config Loader  │
│  (config.go)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────────┐
│  File Parser    │──────▶│  Jira API Client │
│  (parser.go)    │      │  (jira.go)       │
└─────────────────┘      └──────────────────┘
         │
         ▼
┌─────────────────┐
│  Date Calculator│
│  (dates.go)     │
└─────────────────┘
```

### File Structure

```
hours/
├── main.go              # CLI entry point
├── internal/
│   ├── config/
│   │   └── config.go    # Configuration management
│   ├── parser/
│   │   └── parser.go    # File parsing logic
│   ├── jira/
│   │   └── client.go    # Jira API client
│   └── dates/
│       └── calculator.go # Date calculation utilities
├── configs/
│   └── projects.yaml    # Project to ticket mapping
├── data/
│   └── YYYY-MM.yaml     # Monthly time logs
└── docs/
    └── DESIGN.md        # This file
```

## File Formats

### Project Mapping (configs/projects.yaml)

```yaml
projects:
  NPU:
    ticket: NPU-25
    description: Master Epic for Tracking AI/NPU work
  MGNT:
    ticket: BCN-13815
    description: Management Cloud
  MIG:
    ticket: BCN-17538
    description: Master Epic BSNEE/BSN.com Migration to BSN.cloud
  SEC:
    ticket: BCN-17640
    description: Manage and Correct all Security Issues
  PROV:
    ticket: BCN-17540
    description: Refactor Provisioning
  AT:
    ticket: QE-816
    description: Master Epic roll up all implementation work for fully automated testing
  MOB:
    ticket: BCN-17963
    description: Master Epic for Refresh of BrightSign Mobile Application
  BOS:
    ticket: OS-16156
    description: Moka/Walmart work
  HWC:
    ticket: PE-203
    description: Meta HW Compute
  THOR:
    ticket: OS-17165
    description: Thor OS Phase 2 Epic
  CLOUD:
    ticket: BCN-16939
    description: Security/SOC2 Related Work
  OAUTH:
    ticket: BCN-15686
    description: Authentication Phase 1 SSO support
```

### Monthly Time Log (data/2026-01.yaml)

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
  - code: MIG
    week1: 0
    week2: 8
    week3: 2
    week4: 2
  - code: SEC
    week1: 2
    week2: 2
    week3: 2
    week4: 0
  - code: PROV
    week1: 3
    week2: 8
    week3: 5
    week4: 4
  - code: AT
    week1: 0
    week2: 0
    week3: 0
    week4: 0
  - code: MOB
    week1: 0
    week2: 0
    week3: 0
    week4: 2
  - code: BOS
    week1: 0
    week2: 0
    week3: 0
    week4: 0
  - code: HWC
    week1: 0
    week2: 0
    week3: 0
    week4: 0
  - code: THOR
    week1: 1
    week2: 1
    week3: 1
    week4: 1
  - code: CLOUD
    week1: 0
    week2: 0
    week3: 0
    week4: 0
  - code: OAUTH
    week1: 0
    week2: 0
    week3: 0
    week4: 0
```

### Credentials (.envrc or .env)

Required environment variables:

```bash
export JIRA_CLOUD_ID="29ad0f88-9969-4673-b232-4aa64e95f11b"
export JIRA_EMAIL="your-email@brightsign.biz"
export JIRA_TOKEN="your-api-token-here"

# Optional (defaults to https://api.atlassian.com)
export JIRA_BASE_URL="https://api.atlassian.com"
```

## Jira API Integration

### Authentication

Use Basic Auth with email and API token:
```
Authorization: Basic base64(email:api_token)
```

### Add Worklog Endpoint

```
POST /ex/jira/{cloudId}/rest/api/3/issue/{issueIdOrKey}/worklog
```

Request body:
```json
{
  "timeSpent": "2h",
  "started": "2026-01-13T08:00:00.000-0800"
}
```

### Response Handling

- 201 Created: Success
- 400 Bad Request: Invalid input
- 401 Unauthorized: Authentication failure
- 404 Not Found: Issue doesn't exist

### Idempotent Add - Check Before Logging

Before adding a worklog, check if it already exists to prevent duplicates.

#### Algorithm

1. **GET existing worklogs** for the issue
2. **Parse and filter:**
   - Filter by author email (only worklogs from authenticated user)
   - Filter by date (same day as calculated Monday)
   - Check if hours match what we want to log

3. **Decision:**
   - If exact match exists: Skip (report as "already logged")
   - If no match: Proceed with POST to add worklog

#### Date Matching Logic

```go
// Parse worklog started date
worklogDate, _ := time.Parse(time.RFC3339, worklog.Started)

// Compare dates (ignore time component)
targetYear, targetMonth, targetDay := targetMonday.Date()
worklogYear, worklogMonth, worklogDay := worklogDate.Date()

dateMatches := (targetYear == worklogYear &&
                targetMonth == worklogMonth &&
                targetDay == worklogDay)
```

#### Benefits

- **Safe re-runs**: Can run log command multiple times without creating duplicates
- **Incremental updates**: Add new entries without duplicating existing ones
- **Clear reporting**: Shows which entries were skipped vs newly added

### Delete Worklog Support

For error correction when hours are logged incorrectly.

#### Get Worklogs Endpoint

```
GET /ex/jira/{cloudId}/rest/api/3/issue/{issueIdOrKey}/worklog
```

Response includes array of worklogs:
```json
{
  "worklogs": [
    {
      "id": "36643",
      "author": {
        "accountId": "6111da784e8d8d0069dc7889",
        "emailAddress": "gherlein@brightsign.biz"
      },
      "started": "2026-01-05T08:00:00.000-0800",
      "timeSpent": "12h",
      "timeSpentSeconds": 43200
    }
  ]
}
```

#### Delete Worklog Endpoint

```
DELETE /ex/jira/{cloudId}/rest/api/3/issue/{issueIdOrKey}/worklog/{worklogId}
```

Response handling:
- 204 No Content: Success
- 400 Bad Request: Invalid worklog ID
- 401 Unauthorized: Authentication failure
- 404 Not Found: Issue or worklog doesn't exist

#### Delete Algorithm

When `--delete` flag is used:

1. **For each project/week entry:**
   - Calculate the Monday date for that week
   - GET worklogs for the ticket
   - Filter worklogs by:
     - Author matches authenticated user (by email)
     - Started date matches calculated Monday (same day)

2. **For each matching worklog:**
   - Display: `Found worklog {id}: {hours}h on {date}`
   - DELETE the worklog
   - Report success/failure

3. **Handle edge cases:**
   - Multiple worklogs on same date: Delete all from authenticated user
   - No matching worklogs: Report "no worklog found to delete"
   - Partial date match: Match by date only (ignore time component)

#### Delete Mode Flags

```bash
# Delete all logged hours for a month
jira-hours log --month 2026-01 --delete

# Delete specific week only
jira-hours log --month 2026-01 --week 3 --delete

# Delete specific project only
jira-hours log --month 2026-01 --code NPU --delete

# Dry run (show what would be deleted)
jira-hours log --month 2026-01 --delete --dry-run
```

#### Delete Output Example

```
Deleting Hours: 2026-01
========================

Week 1 (Monday, 2026-01-05):
  NPU-25 (NPU): Found worklog 36643: 12h
    ✓ Deleted worklog 36643
  BCN-13815 (MGNT): Found worklog 36644: 24h
    ✓ Deleted worklog 36644
  BCN-17640 (SEC): No worklog found (skipped)
  ...

Week 2 (Monday, 2026-01-12):
  ...

Summary:
========
  Deleted: 15 worklogs
  Not found: 8 entries
  Failed: 0
```

#### Safety Considerations

1. **User Confirmation**
   - Require explicit `--delete` flag
   - Show what will be deleted in dry-run first
   - Consider adding `--confirm` flag for extra safety

2. **Filter by Author**
   - Only delete worklogs created by authenticated user
   - Never delete worklogs from other users
   - Verify author email matches `JIRA_EMAIL`

3. **Date Matching**
   - Match by date only (YYYY-MM-DD)
   - Ignore time component to handle timezone differences
   - Consider worklogs within ±12 hours of target Monday

4. **Error Recovery**
   - Continue on errors (don't stop mid-deletion)
   - Report all failures at end
   - Provide worklog IDs in error messages for manual cleanup

#### Implementation Notes

**Data Structures:**
```go
type Worklog struct {
    ID               string `json:"id"`
    IssueID          string `json:"issueId"`
    Author           Author `json:"author"`
    Started          string `json:"started"`
    TimeSpent        string `json:"timeSpent"`
    TimeSpentSeconds int    `json:"timeSpentSeconds"`
}

type Author struct {
    AccountID    string `json:"accountId"`
    EmailAddress string `json:"emailAddress"`
    DisplayName  string `json:"displayName"`
}

type WorklogsResponse struct {
    Worklogs []Worklog `json:"worklogs"`
    MaxResults int     `json:"maxResults"`
    Total      int     `json:"total"`
}
```

**Core Functions:**
```go
// Get all worklogs for an issue
func (c *Client) GetWorklogs(issueKey string) ([]Worklog, error)

// Delete a specific worklog
func (c *Client) DeleteWorklog(issueKey, worklogID string) error

// Find worklogs matching date and user
func FindMatchingWorklogs(worklogs []Worklog, targetDate time.Time, userEmail string) []Worklog

// Delete worklogs for a month
func DeleteMonth(monthLog *MonthlyLog, projectConfig *ProjectConfig, client *JiraClient, dryRun bool) error
```

#### Use Cases

**Scenario 1: Incorrect hours logged**
- Logged 10h instead of 12h for NPU week 1
- Solution: Delete and re-log
```bash
./bin/jira-hours log --month 2026-01 --week 1 --code NPU --delete
./bin/jira-hours log --month 2026-01 --week 1 --code NPU
```

**Scenario 2: Wrong week**
- Logged hours to week 2 instead of week 3
- Solution: Delete week 2, log to week 3
```bash
./bin/jira-hours log --month 2026-01 --week 2 --delete
# Fix data file
./bin/jira-hours log --month 2026-01 --week 3
```

**Scenario 3: Complete month redo**
- Need to recalculate entire month
- Solution: Delete all, re-log
```bash
./bin/jira-hours log --month 2026-01 --delete --dry-run  # Check first
./bin/jira-hours log --month 2026-01 --delete
# Fix data file
./bin/jira-hours log --month 2026-01
```

**Scenario 4: Accidental double-logging**
- Ran the log command twice by mistake
- Solution: Delete duplicates
```bash
./bin/jira-hours log --month 2026-01 --delete
./bin/jira-hours log --month 2026-01
```

#### Note

See [DELETE-FEATURE.md](DELETE-FEATURE.md) for complete implementation details including:
- API endpoints and request/response formats
- Detailed algorithm and data structures
- Error handling and edge cases
- Testing strategy
- Safety considerations

#### Future Enhancement: Smart Update

Instead of delete + re-add, implement update:

```bash
# Update existing worklogs if they exist, add if they don't
jira-hours log --month 2026-01 --update

# Compare local hours vs Jira worklogs, sync differences
jira-hours sync --month 2026-01
```

## Date Calculation Logic

### Algorithm

Given a month (YYYY-MM) and week number (1-4):

1. Parse month as `time.Time` (first day of month)
2. Find the first Monday of the month:
   - If month starts on Monday, that's week 1
   - Otherwise, advance to next Monday
3. Add (weekNumber - 1) * 7 days to get target Monday
4. Format as ISO 8601 with timezone

### Example

For January 2026:
- January 1, 2026 is a Thursday
- First Monday: January 5, 2026 (Week 1)
- Second Monday: January 12, 2026 (Week 2)
- Third Monday: January 19, 2026 (Week 3)
- Fourth Monday: January 26, 2026 (Week 4)

### Edge Cases

- Month with 5 Mondays: Only process weeks 1-4
- Partial weeks: First Monday might be after day 7
- Timezone handling: Use PST/PDT for consistency

## CLI Interface

### Commands

```bash
# Log all hours for a specific month
hours log --month 2026-01

# Log hours for a specific week
hours log --month 2026-01 --week 3

# Log hours for a specific project code
hours log --month 2026-01 --code NPU

# Dry run (show what would be logged)
hours log --month 2026-01 --dry-run

# Validate time log file
hours validate --month 2026-01

# Show summary
hours summary --month 2026-01
```

### Flags

- `--month`: Month in YYYY-MM format
- `--week`: Week number (1-4, 0 for all weeks)
- `--code`: Capitalization code filter
- `--dry-run`: Show operations without executing
- `--delete` or `-d`: Delete mode (remove worklogs instead of adding)
- `--mock`: Use mock client (no API calls)
- `--timezone`: Timezone for date calculations (default: America/Los_Angeles)

## Implementation Details

### Idempotent Add Behavior

Before adding a worklog, the tool checks if an identical entry already exists:

1. **Get existing worklogs** for the ticket
2. **Filter by:**
   - Author matches authenticated user (by email)
   - Started date matches calculated Monday (same day)
   - Time spent matches hours to be logged

3. **If exact match found:**
   - Skip adding (report as "already logged")
   - Prevents double-logging if command runs twice

4. **If no match found:**
   - Add the worklog normally

This makes the log operation idempotent - running it multiple times with the same data won't create duplicate entries.

**Example Output:**
```
Week 1 (Monday, 2026-01-05):
  ○ NPU-25 (NPU): 12h already logged (worklog 36643)
  ✓ BCN-13815 (MGNT): 24h logged (new)
```

**Implementation:**
```go
// Check if worklog already exists
func (c *Client) WorklogExists(issueKey string, targetDate time.Time, hours int, userEmail string) (bool, string, error)

// Modified add logic:
exists, worklogID, err := client.WorklogExists(issueKey, monday, hours, userEmail)
if err != nil {
    return fmt.Errorf("checking existing worklogs: %w", err)
}
if exists {
    fmt.Printf("  ○ %-10s (%-6s): %dh already logged (worklog %s)\n", ...)
    stats.alreadyLogged++
    continue
}
// Proceed with AddWorklog...
```

## Data Structures

```go
type Project struct {
    Code        string `yaml:"code"`
    Ticket      string `yaml:"ticket"`
    Description string `yaml:"description"`
}

type ProjectConfig struct {
    Projects map[string]Project `yaml:"projects"`
}

type WeekHours struct {
    Code  string `yaml:"code"`
    Week1 int    `yaml:"week1"`
    Week2 int    `yaml:"week2"`
    Week3 int    `yaml:"week3"`
    Week4 int    `yaml:"week4"`
}

type MonthlyLog struct {
    Month string       `yaml:"month"`
    Hours []WeekHours  `yaml:"hours"`
}

type JiraCredentials struct {
    CloudID  string `yaml:"cloud_id"`
    BaseURL  string `yaml:"base_url"`
    Email    string `yaml:"email"`
    APIToken string `yaml:"api_token"`
}

type WorklogEntry struct {
    TimeSpent string `json:"timeSpent"`
    Started   string `json:"started"`
}
```

### Core Functions

```go
// Calculate Monday of week N in given month
func GetWeekMonday(year, month, weekNumber int, timezone string) time.Time

// Parse monthly time log YAML
func ParseMonthlyLog(filepath string) (*MonthlyLog, error)

// Parse project configuration
func ParseProjectConfig(filepath string) (*ProjectConfig, error)

// Create Jira API client
func NewJiraClient(creds JiraCredentials) *JiraClient

// Add worklog to Jira issue
func (c *JiraClient) AddWorklog(issueKey string, hours int, startDate time.Time) error

// Process entire month
func ProcessMonth(monthLog *MonthlyLog, projectConfig *ProjectConfig, client *JiraClient, dryRun bool) error
```

### Validation Rules

1. **Time Log Validation**
   - Month format must be YYYY-MM
   - Week hours must be non-negative integers
   - All codes must exist in project config
   - Total weekly hours should not exceed reasonable limits (e.g., 60h/week)

2. **Project Config Validation**
   - All ticket IDs must be valid format (e.g., QE-816, BCN-13815)
   - No duplicate codes
   - No duplicate ticket IDs

3. **API Validation**
   - Credentials must be present
   - Test connection before bulk operations
   - Verify each ticket exists before logging

### Error Handling

```go
type LogError struct {
    Code    string
    Week    int
    Ticket  string
    Message string
    Err     error
}

// Continue processing on errors but collect all failures
// Display summary at end with all errors
```

## Workflow

### Typical Usage

1. User fills in monthly time log: `data/2026-01.yaml`
2. User runs validation: `hours validate --month 2026-01`
3. User runs dry-run: `hours log --month 2026-01 --dry-run`
4. Review output showing what will be logged
5. User runs actual log: `hours log --month 2026-01`
6. Program logs all non-zero hours to corresponding tickets

### Dry Run Output Example

```
Dry Run: January 2026
=====================

Week 1 (Monday, 2026-01-05):
  ✓ QE-816   (AT)    : 0h (skipped - zero hours)
  ✓ NPU-25   (NPU)   : 12h
  ✓ BCN-13815 (MGNT) : 24h
  ...

Week 2 (Monday, 2026-01-12):
  ✓ QE-816   (AT)    : 0h (skipped - zero hours)
  ✓ NPU-25   (NPU)   : 13h
  ...

Summary:
  Total entries: 48
  Will log: 12
  Will skip: 36 (zero hours)
  Total hours: 149h
```

### Actual Run Output Example

```
Logging Hours: January 2026
===========================

Week 1 (Monday, 2026-01-05):
  ✓ NPU-25   (NPU)   : 12h logged
  ✓ BCN-13815 (MGNT) : 24h logged
  ✗ MIG-17538 (MIG)  : 0h failed (HTTP 404: Issue not found)
  ...

Summary:
  Success: 11
  Failed: 1
  Skipped: 36

Errors:
  - MIG-17538: Issue not found (verify ticket exists)
```

## Configuration Management

### Environment Variables

Environment variables required:
```bash
export JIRA_CLOUD_ID="29ad0f88-9969-4673-b232-4aa64e95f11b"
export JIRA_EMAIL="gherlein@brightsign.biz"
export JIRA_TOKEN="xxxx"
```

### Config Precedence

All credentials must be provided via environment variables (.envrc or .env).
No config file support.

## Migration from Current Format

### Conversion Tool

```bash
# Convert existing markdown to YAML
hours convert --input 202601.md --output data/2026-01.yaml

# Convert projects.md to projects.yaml
hours convert --input PROJECTS.md --output configs/projects.yaml
```

### Backward Compatibility

- Keep markdown files as documentation/reference
- Generate markdown reports from YAML data
- Support both formats for transition period

## Testing Strategy

### Unit Tests

- Date calculation for various month/year combinations
- YAML parsing with valid/invalid inputs
- Ticket ID validation
- Time format conversion

### Integration Tests

- Mock Jira API responses
- End-to-end workflow with test data
- Error handling scenarios

### Test Data

```yaml
# test-data/2026-01-test.yaml
month: 2026-01
hours:
  - code: NPU
    week1: 5
    week2: 0
    week3: 10
    week4: 15
```

## Security Considerations

1. **Credentials Storage**
   - Never commit .envrc or .env files
   - Keep .envrc and .env in .gitignore
   - Use environment variables only (no config files)
   - Validate credentials before bulk operations

2. **API Token Management**
   - Use Jira API tokens, not passwords
   - Document token creation process
   - Support token rotation

3. **Input Validation**
   - Sanitize all user inputs
   - Validate ticket IDs against expected format
   - Limit hour values to reasonable ranges

## Dependencies

```go
// Go modules needed
require (
    gopkg.in/yaml.v3              // YAML parsing
    github.com/spf13/cobra        // CLI framework
    github.com/spf13/viper        // Configuration management
)
```

## Build and Deployment

### Building

```bash
go build -o bin/hours cmd/hours/main.go
```

### Installation

```bash
# Install to $GOPATH/bin
go install ./cmd/hours

# Or use Makefile
make install
```

### Distribution

```bash
# Build for multiple platforms
make build-all

# Creates:
#   dist/hours-darwin-amd64
#   dist/hours-darwin-arm64
#   dist/hours-linux-amd64
#   dist/hours-windows-amd64.exe
```

## Future Enhancements

### Phase 2 Features

1. **Report Generation**
   - Generate markdown reports from Jira worklogs
   - Compare logged vs. planned hours
   - Identify discrepancies

2. **Interactive Mode**
   - Prompt for each entry before logging
   - Allow editing on-the-fly
   - Confirm before API calls

3. **Sync Command**
   - Pull existing worklogs from Jira
   - Compare with local time logs
   - Detect and resolve conflicts

4. **Analytics**
   - Show time distribution across projects
   - Track trends over months
   - Generate charts/graphs

5. **Batch Operations**
   - Delete worklogs for a date range
   - Update existing worklogs
   - Transfer hours between tickets

## Error Recovery

### Handling Partial Failures

If logging fails mid-month:
1. Save state of successfully logged entries
2. Provide resume capability
3. Skip already-logged entries on retry

### State File

```yaml
# .state/2026-01-log-state.yaml
month: 2026-01
logged:
  - code: NPU
    week: 1
    logged_at: 2026-02-01T10:30:00Z
    worklog_id: 36643
  - code: MGNT
    week: 1
    logged_at: 2026-02-01T10:30:15Z
    worklog_id: 36644
```

## API Rate Limiting

### Jira Rate Limits

- Cloud: ~100 requests per minute per user
- Implement exponential backoff
- Add delays between requests (100ms default)
- Batch operations when possible

### Retry Logic

```go
type RetryConfig struct {
    MaxRetries int
    InitialDelay time.Duration
    MaxDelay time.Duration
    Multiplier float64
}

// Default: 3 retries, 1s initial, 10s max, 2x multiplier
```

## Logging and Debugging

### Log Levels

- ERROR: Failed operations
- WARN: Skipped entries, validation warnings
- INFO: Normal operations (default)
- DEBUG: API requests/responses

### Debug Mode

```bash
hours log --month 2026-01 --debug --log-file debug.log
```

Output includes:
- Full API requests/responses
- Date calculations
- File parsing details
- State transitions

## Example Usage Session

```bash
# 1. Initialize new month
$ hours init --month 2026-02
Created data/2026-02.yaml from template

# 2. Edit data/2026-02.yaml with time entries

# 3. Validate before logging
$ hours validate --month 2026-02
✓ File format valid
✓ All project codes found
✓ Date calculations OK
✓ Total hours: 152h (reasonable)

# 4. Dry run
$ hours log --month 2026-02 --dry-run
Would log 11 worklogs (skip 37 zero-hour entries)
Total: 152 hours

# 5. Actually log
$ hours log --month 2026-02
Logging hours for February 2026...
✓ NPU-25: 12h logged (week 1)
✓ BCN-13815: 20h logged (week 1)
...
Success: 11/11 worklogs logged

# 6. Verify in Jira or generate report
$ hours report --month 2026-02
Generated report: reports/2026-02.md
```

## Migration Path

### Step 1: Convert Existing Files

```bash
hours convert --input PROJECTS.md --output configs/projects.yaml
hours convert --input 202601.md --output data/2026-01.yaml
```

### Step 2: Validate Conversions

```bash
hours validate --month 2026-01
```

### Step 3: Test with Dry Run

```bash
hours log --month 2026-01 --dry-run
```

### Step 4: Log to Jira

```bash
hours log --month 2026-01
```

## Alternative: Keep Markdown Format

If preferred, the parser can handle markdown tables directly:

```go
// Use regex or table parser library
func ParseMarkdownTable(content string) (*MonthlyLog, error) {
    // Parse | CAP Code | Week 1 | Week 2 | Week 3 | Week 4 |
    // Extract rows, trim whitespace, convert to struct
}
```

Trade-offs:
- **Markdown**: Human-readable, harder to parse, error-prone
- **YAML**: Machine-readable, easier to parse, requires conversion

Recommendation: Start with YAML, provide conversion tool for existing markdown.

## Open Questions

1. **Timezone**: Always use PST/PDT, or configure per user?
2. **Worklog comments**: Add comment field to describe what was worked on?
3. **Overwrite behavior**: If worklog exists for that date, skip or overwrite?
4. **Week 5**: How to handle months with 5 Mondays?
5. **Partial hours**: Support 0.5h increments or round to nearest hour?
6. **Batch size**: Log all at once or in smaller batches?

## Decision Log

### Format Choice: YAML vs Markdown

**Decision**: Use YAML for data files, provide conversion from markdown

**Rationale**:
- YAML is easier to parse programmatically
- Less error-prone than regex-based markdown parsing
- Better type safety
- Can still generate markdown reports for humans

### Week Definition: Calendar vs Custom

**Decision**: Use calendar weeks starting with first Monday of month

**Rationale**:
- Matches common business week definition
- Deterministic calculation
- No ambiguity about week boundaries

### Error Handling: Fail-fast vs Continue

**Decision**: Continue on errors, report all failures at end

**Rationale**:
- User can fix multiple issues in one pass
- Don't waste successful API calls
- Provide complete error report for troubleshooting
