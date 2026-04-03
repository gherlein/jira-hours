# Jira Hours Tool (jh) - Complete Requirements Specification

## Executive Summary

Jira Hours Tool (binary name: `jh`) is a Go CLI application for bidirectional synchronization of time logging data with Jira Cloud. It reads and writes monthly time logs, maps project codes to Jira tickets, and automates the tedious process of logging hours via the Jira REST API.

## Binary Name

The compiled binary MUST be named `jh` (short for "jira-hours").

## Core Capabilities

1. **Log hours TO Jira** - Write monthly time logs to Jira worklogs
2. **Read hours FROM Jira** - Extract logged hours for a month/user into editable format
3. **Validate time logs** - Check format, project codes, and date calculations
4. **Idempotent operations** - Safe to run multiple times without duplicates
5. **Delete mode** - Remove incorrectly logged hours for corrections
6. **Selective operations** - Filter by week number or project code
7. **Dry-run mode** - Preview operations without executing

## System Architecture

### Directory Structure

```
jira-hours/
├── cmd/hours/
│   ├── main.go           # CLI entry point with cobra commands
│   ├── log.go            # Log command (add/delete worklogs)
│   ├── validate.go       # Validate command
│   └── fetch.go          # NEW: Fetch command (read from Jira)
├── internal/
│   ├── config/
│   │   └── config.go     # Config and credentials loading
│   ├── parser/
│   │   └── parser.go     # YAML parsing for monthly logs
│   ├── jira/
│   │   └── client.go     # Jira API client implementation
│   └── dates/
│       └── calculator.go # Date calculation utilities
├── configs/
│   └── projects.yaml     # Project code to Jira ticket mapping
├── data/
│   └── YYYY-MM.yaml      # Monthly time logs
├── go.mod                # Go module definition
├── Makefile              # Build automation
└── README.md             # User documentation
```

### Language and Framework

- **Language**: Go 1.21 or later
- **CLI Framework**: github.com/spf13/cobra
- **YAML Parser**: gopkg.in/yaml.v3
- **HTTP Client**: net/http (stdlib)
- **Testing**: stdlib testing package

### Build System

Makefile targets (run `make` with no arguments to print all targets):

```makefile
all          - Download dependencies and build
build        - Build the jh binary to bin/jh
install      - Install jh to GOPATH/bin
test         - Run Go unit tests
clean        - Remove build artifacts
validate-test - Validate test data file
run-test     - Run with mock client
dry-run      - Show what would be logged
fmt          - Format Go code
vet          - Run go vet
check        - Run fmt, vet, and test
```

Build command:
```bash
go build -o bin/jh ./cmd/hours
```

## Data Formats

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
  - code: MIG
    week1: 0
    week2: 8
    week3: 2
    week4: 2
```

**Format Rules**:
- `month` field: YYYY-MM format (required)
- `hours` array: List of project entries
- Each entry has:
  - `code`: Project code string (maps to configs/projects.yaml)
  - `week1`, `week2`, `week3`, `week4`: Integer hours (0 or positive)
- Zero hours are valid (means no work that week)
- No duplicate codes allowed
- Negative hours are invalid
- Weekly total > 240 hours is considered unreasonable (warning)
- Total monthly hours > 200 triggers warning

**Go Data Structure**:
```go
type WeekHours struct {
    Code  string `yaml:"code"`
    Week1 int    `yaml:"week1"`
    Week2 int    `yaml:"week2"`
    Week3 int    `yaml:"week3"`
    Week4 int    `yaml:"week4"`
}

type MonthlyLog struct {
    Month string      `yaml:"month"`
    Hours []WeekHours `yaml:"hours"`
}
```

### Project Configuration (configs/projects.yaml)

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
```

**Format Rules**:
- `projects` map: Keys are project codes (strings)
- Each project has:
  - `ticket`: Jira ticket ID (e.g., "NPU-25", "BCN-13815")
  - `description`: Human-readable description
- No duplicate ticket IDs allowed
- Ticket IDs must match pattern: [A-Z]+-[0-9]+

**Go Data Structure**:
```go
type Project struct {
    Ticket      string `yaml:"ticket"`
    Description string `yaml:"description"`
}

type ProjectConfig struct {
    Projects map[string]Project `yaml:"projects"`
}
```

### Projects List (PROJECTS.md or configured file)

Markdown table format:

```markdown
| Project | Code | Jira Ticket |
|---------|------|-------------|
| NPU | NPU | https://brightsign.atlassian.net/browse/NPU-25 |
| Management Cloud | MGNT | https://brightsign.atlassian.net/browse/BCN-13815 |
| Migration | MIG | https://brightsign.atlassian.net/browse/BCN-17538 |
```

**Parser Requirements**:
- Extract ticket IDs from URLs or plain text (NPU-25, BCN-13815)
- Support both URL and plain ticket ID formats
- Ignore header row (starts with "Project" or uses |-----|)
- Extract project code from second column
- Extract ticket ID from third column (parse from URL if needed)

## Jira API Integration

### Authentication

**Method**: Basic Authentication with email and API token

```
Authorization: Basic base64(email:api_token)
```

**Required Environment Variables**:
```bash
export JIRA_CLOUD_ID="29ad0f88-9969-4673-b232-4aa64e95f11b"
export JIRA_EMAIL="user@example.com"
export JIRA_TOKEN="api-token-here"
export JIRA_BASE_URL="https://api.atlassian.com"  # Optional, defaults to this
```

**Credentials Loading**:
- Read from environment variables only
- No config file support
- Validate all required fields present before operations
- Test connection before bulk operations

**Go Data Structure**:
```go
type JiraCredentials struct {
    CloudID  string
    BaseURL  string
    Email    string
    APIToken string
}
```

### API Endpoints Used

#### 1. Test Connection
```
GET {baseURL}/ex/jira/{cloudId}/rest/api/3/myself
```
Returns current user info. Used to verify credentials.

#### 2. Add Worklog
```
POST {baseURL}/ex/jira/{cloudId}/rest/api/3/issue/{issueKey}/worklog
Content-Type: application/json

{
  "timeSpent": "12h",
  "started": "2026-01-05T08:00:00.000-0800"
}
```
Response: 201 Created with worklog details

#### 3. Get Worklogs
```
GET {baseURL}/ex/jira/{cloudId}/rest/api/3/issue/{issueKey}/worklog
```
Response: 200 OK with array of worklogs

#### 4. Delete Worklog
```
DELETE {baseURL}/ex/jira/{cloudId}/rest/api/3/issue/{issueKey}/worklog/{worklogId}
```
Response: 204 No Content on success

### Worklog Data Structure

**Jira API Response Format**:
```json
{
  "worklogs": [
    {
      "id": "36643",
      "issueId": "12345",
      "author": {
        "accountId": "6111da784e8d8d0069dc7889",
        "emailAddress": "user@example.com",
        "displayName": "User Name"
      },
      "started": "2026-01-05T08:00:00.000-0800",
      "timeSpent": "12h",
      "timeSpentSeconds": 43200
    }
  ],
  "maxResults": 1048576,
  "total": 1
}
```

**Go Data Structure**:
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
    Worklogs   []Worklog `json:"worklogs"`
    MaxResults int       `json:"maxResults"`
    Total      int       `json:"total"`
}
```

## Date Calculation Algorithm

### Week Definition

- **Week 1**: First Monday of the month at 8:00 AM local time
- **Week 2**: First Monday + 7 days
- **Week 3**: First Monday + 14 days
- **Week 4**: First Monday + 21 days

### Algorithm

```go
func GetWeekMonday(year, month, weekNumber int, timezone string) (time.Time, error) {
    // 1. Load timezone
    loc, _ := time.LoadLocation(timezone)

    // 2. Get first day of month at 8:00 AM
    firstDay := time.Date(year, time.Month(month), 1, 8, 0, 0, 0, loc)

    // 3. Find first Monday
    firstMonday := firstDay
    for firstMonday.Weekday() != time.Monday {
        firstMonday = firstMonday.AddDate(0, 0, 1)
    }

    // 4. Calculate target Monday
    targetMonday := firstMonday.AddDate(0, 0, (weekNumber-1)*7)

    // 5. Validate still in same month
    if targetMonday.Month() != time.Month(month) {
        return time.Time{}, fmt.Errorf("week %d extends beyond month", weekNumber)
    }

    return targetMonday, nil
}
```

### Date Matching for Worklogs

When checking if a worklog matches a target date:

1. Parse worklog `started` field (RFC3339 or "2006-01-02T15:04:05.000-0700")
2. Extract date components only (ignore time)
3. Compare year, month, day (must all match)
4. Ignore time zone differences

```go
func isSameDay(worklogStarted string, targetDate time.Time) bool {
    worklogDate, err := time.Parse(time.RFC3339, worklogStarted)
    if err != nil {
        worklogDate, err = time.Parse("2006-01-02T15:04:05.000-0700", worklogStarted)
        if err != nil {
            return false
        }
    }

    wYear, wMonth, wDay := worklogDate.Date()
    tYear, tMonth, tDay := targetDate.Date()

    return wYear == tYear && wMonth == tMonth && wDay == tDay
}
```

### Example: January 2026

- January 1, 2026 is Thursday
- Week 1: Monday, January 5, 2026
- Week 2: Monday, January 12, 2026
- Week 3: Monday, January 19, 2026
- Week 4: Monday, January 26, 2026

### Timezone Handling

- Default timezone: America/Los_Angeles
- Configurable via `--timezone` flag
- Use `time.LoadLocation()` to load timezone
- All date calculations use specified timezone
- Jira API uses ISO 8601 format with timezone offset

## Command Line Interface

### CLI Framework

Use `github.com/spf13/cobra` for command structure.

### Root Command

```
jh - Jira time logging automation tool

Usage:
  jh [command]

Available Commands:
  log         Log or delete hours in Jira
  validate    Validate time log files
  fetch       Fetch logged hours from Jira (NEW)
  help        Help about any command
  version     Print version information

Flags:
  -h, --help      Help for jh
  -v, --version   Version information
```

### Command 1: log (Add or Delete Worklogs)

```bash
jh log --month YYYY-MM [flags]
```

**Purpose**: Log hours from local YAML file to Jira, or delete existing worklogs.

**Required Flags**:
- `--month, -m`: Month in YYYY-MM format (required)

**Optional Flags**:
- `--week, -w`: Week number (1-4, 0 for all weeks, default: 0)
- `--code, -c`: Project code filter (default: all projects)
- `--dry-run`: Show what would be logged without API calls (default: false)
- `--delete, -d`: Delete mode - remove worklogs instead of adding (default: false)
- `--mock`: Use mock client for testing (default: false)
- `--timezone, -t`: Timezone for date calculations (default: "America/Los_Angeles")

**Behavior**:

1. **Load data**:
   - Read `data/{month}.yaml`
   - Read `configs/projects.yaml`
   - Load credentials from environment

2. **Test connection** (unless mock or dry-run)

3. **For each week** (filtered by --week if specified):
   - Calculate Monday date
   - For each project (filtered by --code if specified):
     - Get hours for that week
     - Skip if hours = 0
     - Map code to ticket ID
     - **Add mode**:
       - Check if identical worklog exists (idempotent)
       - Skip if already logged
       - Add worklog if new
     - **Delete mode** (--delete flag):
       - Find worklogs matching date and user
       - Delete each matching worklog
       - Skip if no matches found

4. **Print summary**:
   - Success count / Already logged count / Failed count
   - Total hours processed
   - List of errors if any

**Add Mode Output Example**:
```
Logging Hours: 2026-01
======================

Week 1 (Monday, 2026-01-05):
  ✓ NPU-25   (NPU)   : 12h logged
  ○ BCN-13815 (MGNT) : 24h already logged (worklog 36643)
  - MIG-123  (MIG)   : 0h (skipped)
  ✗ SEC-456  (SEC)   : 2h failed: Issue not found

Summary:
========
  Success: 1 new
  Already logged: 1 (skipped)
  Failed: 1
  Skipped: 1 (zero hours)
  Total hours: 38
```

**Delete Mode Output Example**:
```
Deleting Hours: 2026-01
========================

Week 1 (Monday, 2026-01-05):
  ✓ NPU-25   (NPU)   : deleted worklog 36643 (12h)
  - BCN-13815 (MGNT) : no worklog found (skipped)

Summary:
========
  Deleted: 1 worklogs
  Not found: 1 entries
  Total hours: 12
```

**Idempotent Add Behavior**:
- Before adding a worklog, check if identical entry exists
- Match by: same issue, same date (day only), same hours, same author email
- If match found: Skip and report "already logged" with worklog ID
- If no match: Add new worklog
- This allows safe re-runs without creating duplicates

**Error Handling**:
- Continue processing on errors (don't fail-fast)
- Collect all errors and display at end
- Return non-zero exit code if any failures
- Provide specific error messages (API errors, unknown codes, etc.)

### Command 2: validate (Validate Time Logs)

```bash
jh validate --month YYYY-MM [flags]
```

**Purpose**: Validate monthly time log and project configuration without making API calls.

**Required Flags**:
- `--month, -m`: Month in YYYY-MM format (required)

**Optional Flags**:
- `--timezone, -t`: Timezone for date calculations (default: "America/Los_Angeles")

**Behavior**:

1. Parse month format (YYYY-MM)
2. Load `data/{month}.yaml`
3. Load `configs/projects.yaml`
4. Validate all project codes exist in config
5. Calculate Monday dates for all 4 weeks
6. Display hours summary (per project and total)
7. Warn if total > 200 hours

**Output Example**:
```
Validating: 2026-01
===================

✓ Month format valid
✓ Time log file valid
✓ Project config valid
✓ All project codes found
✓ Date calculations OK

Hours Summary:
--------------
  NPU-25 (NPU)        :  47 hours (W1:12 W2:13 W3:10 W4:12)
  BCN-13815 (MGNT)    :  58 hours (W1:24 W2:10 W3:11 W4:13)
  BCN-17538 (MIG)     :  12 hours (W1: 0 W2: 8 W3: 2 W4: 2)

Total: 149 hours across 11 projects

✓ Validation passed!
```

### Command 3: fetch (NEW - Fetch Hours from Jira)

```bash
jh fetch --month YYYY-MM --user EMAIL [flags]
```

**Purpose**: Read worklogs from Jira for a specified month and user, write to an editable YAML file.

**Required Flags**:
- `--month, -m`: Month in YYYY-MM format (required)
- `--user, -u`: User email address to filter worklogs (required)

**Optional Flags**:
- `--output, -o`: Output file path (default: `data/{month}-fetched.yaml`)
- `--projects-file`: Path to projects list file (default: `./PROJECTS.md`)
- `--config-projects`: Use projects file from user config (default: false)
- `--timezone, -t`: Timezone for date calculations (default: "America/Los_Angeles")
- `--dry-run`: Show what would be fetched without writing file (default: false)

**Projects File Location Priority**:

1. If `--projects-file` specified: Use that exact path
2. If `--config-projects` flag set: Use `~/.config/jh/projects.md`
3. Default: Use `./PROJECTS.md` in current directory

**Behavior**:

1. **Load credentials** from environment
2. **Test connection** to Jira
3. **Determine projects file**:
   - Check flags for explicit path or config location
   - Load and parse projects list (PROJECTS.md format)
   - Extract project codes and ticket IDs
4. **Calculate week dates** for the month
5. **For each project/ticket**:
   - Fetch worklogs from Jira for that ticket
   - Filter worklogs by user email (--user flag)
   - For each of 4 weeks:
     - Find worklogs matching that week's Monday date
     - Sum hours for that week
6. **Generate YAML file**:
   - Same format as monthly time log (data/YYYY-MM.yaml)
   - Include all projects from projects list
   - Hours are summed for each week
   - Zero hours if no worklogs found
7. **Write file** to output path
8. **Display summary** of fetched data

**Output Example**:
```
Fetching Hours from Jira: 2026-01
User: user@example.com
==================================

Loading projects from: ./PROJECTS.md
Found 11 projects

Week 1 (Monday, 2026-01-05):
  NPU-25   (NPU)   : 12h
  BCN-13815 (MGNT) : 24h
  BCN-17538 (MIG)  : 0h
  ...

Week 2 (Monday, 2026-01-12):
  NPU-25   (NPU)   : 13h
  BCN-13815 (MGNT) : 10h
  ...

Summary:
========
  Tickets fetched: 11
  Total worklogs: 48
  Total hours: 149
  Written to: data/2026-01-fetched.yaml

To submit changes:
  1. Edit data/2026-01-fetched.yaml
  2. Run: jh log --month 2026-01 --delete
  3. Run: jh log --month 2026-01
```

**Generated YAML File Format**:

Same format as manual monthly logs:

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
```

**Editing Workflow**:

1. User runs: `jh fetch --month 2026-01 --user user@example.com`
2. File generated: `data/2026-01-fetched.yaml`
3. User edits file to correct hours
4. User deletes old worklogs: `jh log --month 2026-01 --delete --dry-run` (verify first)
5. User deletes old worklogs: `jh log --month 2026-01 --delete`
6. User renames file: `mv data/2026-01-fetched.yaml data/2026-01.yaml`
7. User re-logs corrected hours: `jh log --month 2026-01`

**Hour Aggregation Logic**:

- Multiple worklogs on same day: Sum the hours
- Worklogs spanning multiple days: Count for the day they started
- Partial hours: Round to nearest integer (Jira timeSpentSeconds / 3600)
- Only include worklogs from specified user (filter by author.emailAddress)

**Error Handling**:

- Ticket not found: Log warning, continue with other tickets
- No worklogs for ticket: Include in output with zeros
- API errors: Display error, skip that ticket
- Invalid projects file: Fail with clear error message

## Go Package Structure

### internal/config/config.go

**Functions**:
```go
func LoadProjectConfig(filepath string) (*ProjectConfig, error)
func LoadCredentialsFromEnv() (*JiraCredentials, error)
func validateProjectConfig(config *ProjectConfig) error
func validateCredentials(creds *JiraCredentials) error
```

**Validation Rules**:
- Projects config: No duplicate codes, no duplicate tickets, all tickets valid format
- Credentials: All required env vars present (CloudID, Email, Token)

### internal/parser/parser.go

**Functions**:
```go
func ParseMonthlyLog(filepath string) (*MonthlyLog, error)
func (log *MonthlyLog) GetTotalHours() int
func (log *MonthlyLog) GetHoursForWeek(week int) map[string]int
func (wh *WeekHours) GetWeekHours(week int) int
func validateMonthlyLog(log *MonthlyLog) error
```

**NEW Functions for fetch command**:
```go
func ParseProjectsList(filepath string) (map[string]string, error)
func WriteMonthlyLog(filepath string, log *MonthlyLog) error
```

**ParseProjectsList** reads PROJECTS.md and returns map[code]ticket:
- Input: Markdown table with columns: Project | Code | Jira Ticket
- Output: `map[string]string` where key=code, value=ticketID
- Extract ticket ID from URLs or plain text
- Handle both formats: "NPU-25" or "https://jira.com/browse/NPU-25"

**WriteMonthlyLog** writes MonthlyLog struct to YAML file:
- Standard YAML formatting
- Preserve order (month field first, then hours array)
- Proper indentation (2 spaces)

### internal/jira/client.go

**Existing Functions**:
```go
func NewClient(creds *JiraCredentials) *Client
func (c *Client) AddWorklog(issueKey string, hours int, startDate time.Time) error
func (c *Client) TestConnection() error
func (c *Client) GetWorklogs(issueKey string) ([]Worklog, error)
func (c *Client) DeleteWorklog(issueKey, worklogID string) error
func (c *Client) FindMatchingWorklogs(issueKey string, targetDate time.Time, userEmail string) ([]Worklog, error)
func (c *Client) WorklogExists(issueKey string, targetDate time.Time, hours int, userEmail string) (bool, string, error)
```

**NEW Functions for fetch command**:
```go
func (c *Client) FetchMonthWorklogs(issueKeys []string, year, month int, timezone string) (map[string]map[int]int, error)
func (c *Client) GetUserWorklogs(issueKey string, userEmail string) ([]Worklog, error)
```

**FetchMonthWorklogs**:
- Input: List of ticket IDs, year, month, timezone
- Output: map[ticketID]map[weekNumber]hours
- For each ticket: Get all worklogs, filter by month, group by week
- Calculate which week each worklog belongs to based on date
- Sum hours for worklogs on same week

**GetUserWorklogs**:
- Get all worklogs for an issue
- Filter to only those by specified user email
- Return filtered list

### internal/dates/calculator.go

**Existing Functions**:
```go
func GetWeekMonday(year, month, weekNumber int, timezone string) (time.Time, error)
func ParseMonth(monthStr string) (year, month int, err error)
func FormatForJira(t time.Time) string
```

**NEW Functions**:
```go
func GetWeekNumber(date time.Time, year, month int, timezone string) (int, error)
```

**GetWeekNumber**:
- Given a specific date and the month context
- Determine which week (1-4) that date falls in
- Compare against the 4 Monday dates for that month
- Return week number (1-4) or error if outside month

### cmd/hours/main.go

**Root Command Setup**:
```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "jh",
        Short: "Jira time logging automation tool",
        Long:  "A CLI tool to automate logging hours between monthly time sheets and Jira tickets",
        Version: version,
    }

    rootCmd.AddCommand(newLogCmd())
    rootCmd.AddCommand(newValidateCmd())
    rootCmd.AddCommand(newFetchCmd())

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### cmd/hours/fetch.go (NEW)

**Command Structure**:
```go
func newFetchCmd() *cobra.Command {
    var month string
    var user string
    var output string
    var projectsFile string
    var configProjects bool
    var timezone string
    var dryRun bool

    cmd := &cobra.Command{
        Use:   "fetch",
        Short: "Fetch logged hours from Jira",
        Long:  "Read worklogs from Jira for a specified month and user, write to editable YAML file",
        RunE: func(cmd *cobra.Command, args []string) error {
            return fetchHours(month, user, output, projectsFile, configProjects, timezone, dryRun)
        },
    }

    cmd.Flags().StringVarP(&month, "month", "m", "", "Month in YYYY-MM format")
    cmd.Flags().StringVarP(&user, "user", "u", "", "User email address to filter worklogs")
    cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
    cmd.Flags().StringVar(&projectsFile, "projects-file", "", "Path to projects list file")
    cmd.Flags().BoolVar(&configProjects, "config-projects", false, "Use projects file from user config")
    cmd.Flags().StringVarP(&timezone, "timezone", "t", "America/Los_Angeles", "Timezone for date calculations")
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be fetched without writing file")

    cmd.MarkFlagRequired("month")
    cmd.MarkFlagRequired("user")

    return cmd
}
```

**fetchHours Implementation**:
```go
func fetchHours(month, user, output, projectsFile string, configProjects bool, timezone string, dryRun bool) error {
    // 1. Parse month
    year, monthNum, err := dates.ParseMonth(month)
    if err != nil {
        return err
    }

    // 2. Determine output path
    if output == "" {
        output = filepath.Join("data", month+"-fetched.yaml")
    }

    // 3. Determine projects file path
    projectsPath := getProjectsFilePath(projectsFile, configProjects)

    // 4. Parse projects list
    projectsMap, err := parser.ParseProjectsList(projectsPath)
    if err != nil {
        return fmt.Errorf("loading projects list: %w", err)
    }

    // 5. Load credentials and create client
    creds, err := config.LoadCredentialsFromEnv()
    if err != nil {
        return fmt.Errorf("loading credentials: %w", err)
    }

    client := jira.NewClient(creds)

    // 6. Test connection
    if !dryRun {
        if err := client.TestConnection(); err != nil {
            return fmt.Errorf("jira connection failed: %w", err)
        }
    }

    // 7. Fetch worklogs for each project/week
    monthlyLog := &parser.MonthlyLog{
        Month: month,
        Hours: make([]parser.WeekHours, 0, len(projectsMap)),
    }

    for code, ticket := range projectsMap {
        weekHours := parser.WeekHours{Code: code}

        for week := 1; week <= 4; week++ {
            monday, err := dates.GetWeekMonday(year, monthNum, week, timezone)
            if err != nil {
                // Week extends beyond month, use 0
                continue
            }

            if !dryRun {
                // Get worklogs for this ticket
                worklogs, err := client.GetUserWorklogs(ticket, user)
                if err != nil {
                    fmt.Printf("⚠ %s (%s): error fetching: %v\n", ticket, code, err)
                    continue
                }

                // Sum hours for this week
                hours := 0
                for _, wl := range worklogs {
                    if isSameDay(wl.Started, monday) {
                        hours += wl.TimeSpentSeconds / 3600
                    }
                }

                // Set week hours
                switch week {
                case 1: weekHours.Week1 = hours
                case 2: weekHours.Week2 = hours
                case 3: weekHours.Week3 = hours
                case 4: weekHours.Week4 = hours
                }
            }
        }

        monthlyLog.Hours = append(monthlyLog.Hours, weekHours)
    }

    // 8. Write output file (unless dry-run)
    if !dryRun {
        if err := parser.WriteMonthlyLog(output, monthlyLog); err != nil {
            return fmt.Errorf("writing output file: %w", err)
        }
    }

    // 9. Display summary
    printFetchSummary(monthlyLog, output, dryRun)

    return nil
}

func getProjectsFilePath(projectsFile string, configProjects bool) string {
    if projectsFile != "" {
        return projectsFile
    }
    if configProjects {
        homeDir, _ := os.UserHomeDir()
        return filepath.Join(homeDir, ".config", "jh", "projects.md")
    }
    return "./PROJECTS.md"
}
```

## Mock Client for Testing

**Mock Client** (internal/jira/client.go):
```go
type MockClient struct {
    loggedEntries    []WorklogRequest
    existingWorklogs map[string][]Worklog
    nextWorklogID    int
}

func NewMockClient() *MockClient {
    return &MockClient{
        loggedEntries:    make([]WorklogRequest, 0),
        existingWorklogs: make(map[string][]Worklog),
        nextWorklogID:    1000,
    }
}
```

**Mock Client** implements same interface as real client:
- AddWorklog: Adds to in-memory store
- GetWorklogs: Returns from in-memory store
- DeleteWorklog: Removes from in-memory store
- TestConnection: Always succeeds
- All other methods: Stub implementations

Use mock client with `--mock` flag for testing without API calls.

## Validation Rules

### Time Log Validation
- Month format: YYYY-MM (e.g., "2026-01")
- Hours: Non-negative integers (0 or positive)
- No duplicate project codes
- All project codes must exist in projects.yaml
- Weekly total > 60h generates warning
- Monthly total > 240h generates warning

### Project Config Validation
- No duplicate project codes
- No duplicate ticket IDs
- Ticket IDs match format: [A-Z]+-[0-9]+
- All fields required (code, ticket, description)

### Credentials Validation
- JIRA_CLOUD_ID: Required, non-empty
- JIRA_EMAIL: Required, non-empty, valid email format
- JIRA_TOKEN: Required, non-empty
- JIRA_BASE_URL: Optional, defaults to "https://api.atlassian.com"

## Error Handling Strategy

### General Principles
- Continue processing on errors (don't fail-fast)
- Collect all errors and display summary at end
- Return non-zero exit code if any failures
- Provide specific, actionable error messages

### Error Types

**Configuration Errors** (fail immediately):
- Missing environment variables
- Invalid month format
- File not found (data/config files)
- Invalid YAML syntax

**API Errors** (continue, collect, report):
- Connection failure
- Authentication failure (401)
- Issue not found (404)
- Rate limiting (429)
- Server errors (5xx)

**Validation Errors** (fail before API calls):
- Unknown project codes
- Negative hours
- Invalid date calculations

### Error Output Format
```
✗ NPU-25 (NPU): failed: jira api error (status 404): Issue does not exist
✗ SEC-123 (SEC): failed: jira api error (status 401): Authentication required

Summary:
========
  Success: 8
  Failed: 2

Error: some worklogs failed to log
```

## Testing Strategy

### Unit Tests

**Date Calculations**:
- Test various months/years
- Test months with different start days
- Test edge cases (5 Monday months)
- Test timezone handling

**YAML Parsing**:
- Valid files
- Invalid syntax
- Missing required fields
- Duplicate codes

**Validation**:
- Valid data passes
- Invalid data fails with specific errors
- Edge cases (zero hours, large hours)

**Mock Client**:
- Test all operations with mock
- Verify state changes
- Test idempotent behavior

### Integration Tests

**End-to-End Workflows**:
```bash
# Test with mock client
make run-test

# Test validation
make validate-test

# Test dry-run
make dry-run
```

### Test Data

Maintain test files:
- `data/2026-02.yaml` - Small test data set (2 projects, 4 weeks)
- `configs/projects.yaml` - All project mappings
- Mock client with predefined worklogs for idempotent tests

## Security Requirements

### Credentials
- Never commit credentials to git
- Use environment variables only
- Keep .envrc and .env in .gitignore
- Use Jira API tokens, not passwords

### Input Validation
- Validate all user inputs before API calls
- Sanitize ticket IDs (match expected format)
- Limit hour values (0-240 per week)
- Validate date ranges

### API Communication
- Use HTTPS only
- Verify TLS certificates
- Set reasonable timeouts (30 seconds)
- Handle rate limiting gracefully

## Installation and Distribution

### Installation from Source
```bash
git clone <repo>
cd jira-hours
make all
```

Binary will be at: `bin/jh`

### Installation to PATH
```bash
make install
```

Installs to: `$GOPATH/bin/jh`

### Multi-Platform Builds
```bash
make build-all
```

Creates in `dist/`:
- jh-darwin-amd64 (macOS Intel)
- jh-darwin-arm64 (macOS ARM)
- jh-linux-amd64 (Linux)
- jh-windows-amd64.exe (Windows)

## Configuration Files

### Global Config Directory

User configuration directory: `~/.config/jh/`

Optional files:
- `~/.config/jh/projects.md` - Global projects list (use with --config-projects)

### Project Config Directory

Project-level configuration: `./configs/`

Required:
- `./configs/projects.yaml` - Project code to ticket mappings

### Data Directory

Time log storage: `./data/`

Files:
- `./data/YYYY-MM.yaml` - Monthly time logs for logging
- `./data/YYYY-MM-fetched.yaml` - Fetched data from Jira (editable)

## Usage Examples

### Basic Workflow

```bash
# 1. Create/edit monthly time log
vim data/2026-01.yaml

# 2. Validate before logging
jh validate --month 2026-01

# 3. Preview with dry-run
jh log --month 2026-01 --dry-run

# 4. Log to Jira
jh log --month 2026-01

# 5. Verify (can run again, will be idempotent)
jh log --month 2026-01
```

### Correction Workflow

```bash
# 1. Discover error in logged hours
# 2. Fetch current state from Jira
jh fetch --month 2026-01 --user user@example.com

# 3. Edit fetched file
vim data/2026-01-fetched.yaml

# 4. Delete old worklogs (dry-run first)
jh log --month 2026-01 --delete --dry-run

# 5. Delete old worklogs (execute)
jh log --month 2026-01 --delete

# 6. Rename and re-log
mv data/2026-01-fetched.yaml data/2026-01.yaml
jh log --month 2026-01
```

### Selective Operations

```bash
# Log specific week only
jh log --month 2026-01 --week 3

# Log specific project only
jh log --month 2026-01 --code NPU

# Delete specific week and project
jh log --month 2026-01 --week 2 --code MGNT --delete

# Fetch with custom projects file
jh fetch --month 2026-01 --user user@example.com --projects-file ~/my-projects.md

# Fetch using config directory
jh fetch --month 2026-01 --user user@example.com --config-projects
```

## Output Format Standards

### Success Indicators
- `✓` Green checkmark for success
- `○` Circle for already-logged (idempotent skip)
- `-` Dash for skipped (zero hours)
- `⚠` Warning triangle for warnings
- `✗` Red X for failures

### Alignment
- Ticket IDs: Left-aligned, width 10
- Project codes: Left-aligned in parentheses, width 6
- Hours: Right-aligned with "h" suffix
- Messages: After colon

### Example
```
Week 1 (Monday, 2026-01-05):
  ✓ NPU-25     (NPU)   : 12h logged
  ○ BCN-13815  (MGNT)  : 24h already logged (worklog 36643)
  - MIG-123    (MIG)   : 0h (skipped)
  ✗ SEC-456    (SEC)   : 2h failed: Issue not found
```

## Implementation Priority

### Phase 1: Core Functionality (Existing)
- [x] CLI framework with cobra
- [x] Log command (add worklogs)
- [x] Validate command
- [x] YAML parsing
- [x] Jira API client
- [x] Date calculations
- [x] Idempotent add behavior
- [x] Delete mode
- [x] Mock client
- [x] Build system (Makefile)

### Phase 2: Fetch Command (NEW)
- [ ] Parse PROJECTS.md format
- [ ] Fetch worklogs from Jira
- [ ] Filter by user email
- [ ] Aggregate hours by week
- [ ] Write fetched data to YAML
- [ ] Support projects file from config directory
- [ ] fetch command implementation
- [ ] Integration with existing commands
- [ ] Documentation updates

### Phase 3: Future Enhancements
- Report generation (markdown from Jira)
- Interactive mode (prompt before each operation)
- Sync command (bidirectional comparison)
- Analytics (time distribution, trends)
- State management (resume after failures)
- Rate limiting and retry logic

## Glossary

- **CAP Code**: Project capitalization code (e.g., NPU, MGNT)
- **Ticket**: Jira issue identifier (e.g., NPU-25, BCN-13815)
- **Worklog**: Jira time entry associated with a ticket
- **Week**: Monday of a specific week in a month (Week 1 = first Monday)
- **Idempotent**: Operation that can be repeated safely without duplicate effects
- **Dry-run**: Preview mode that shows operations without executing
- **Mock client**: Test implementation that doesn't make real API calls

## Success Criteria

A complete implementation MUST:

1. Build successfully with `make build` to create `bin/jh`
2. Pass all tests with `make test`
3. Support all three commands: log, validate, fetch
4. Handle environment variable configuration
5. Implement idempotent add behavior
6. Support delete mode for corrections
7. Fetch worklogs from Jira with user filtering
8. Parse PROJECTS.md format correctly
9. Write valid YAML output
10. Provide clear error messages
11. Support dry-run mode for all operations
12. Use mock client for testing
13. Follow date calculation algorithm precisely
14. Support timezone configuration
15. Generate properly formatted output

## Edge Cases and Special Conditions

### Date Edge Cases
- Months starting on Monday (first day is Week 1)
- Months with 5 Mondays (only process weeks 1-4)
- Short months where week 4 extends to next month (error)
- Timezone transitions (DST changes)

### Data Edge Cases
- Zero hours (skip logging, include in fetch output)
- Very large hours (>60/week, warn but allow)
- Missing project codes (error in log, skip in fetch)
- Duplicate worklogs on same date (sum in fetch, detect in add)

### API Edge Cases
- Rate limiting (HTTP 429)
- Temporary network failures
- Partial responses (some tickets succeed, some fail)
- Empty worklog arrays (no time logged for ticket)
- User not found or no access to ticket

### File Edge Cases
- Missing data directory (create it)
- Missing configs directory (error)
- Empty PROJECTS.md (error)
- Malformed YAML (error with line number)
- File permissions (handle gracefully)

## Version History

- **v0.1.0**: Initial implementation (log and validate commands)
- **v0.2.0**: NEW - Add fetch command for reading from Jira

---

## Summary for AI Agents

This specification defines a complete Go CLI tool for bidirectional Jira time logging. The tool can:

1. **Write hours TO Jira** from local YAML files (existing)
2. **Read hours FROM Jira** into editable YAML files (NEW)
3. Validate data before operations
4. Delete worklogs for corrections
5. Operate idempotently (safe to re-run)

Key technical requirements:
- Go 1.21+ with cobra and yaml.v3
- Jira Cloud REST API v3
- Environment variable configuration
- Week-based time tracking (Monday = start of week)
- YAML data format for all time logs
- Markdown projects list parsing (NEW)

The binary name MUST be `jh` and must support:
- `jh log` - Write to Jira
- `jh validate` - Check data
- `jh fetch` - Read from Jira (NEW)

All date calculations use first Monday of month as Week 1, with subsequent weeks at 7-day intervals. The system must support idempotent adds (check before creating) and safe deletions (filter by user email).
