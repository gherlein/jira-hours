# Delete Worklog Feature Design

## Purpose

Enable correction of incorrectly logged hours by deleting worklogs that match specific criteria.

## API Endpoints

### Get Worklogs

```
GET /ex/jira/{cloudId}/rest/api/3/issue/{issueIdOrKey}/worklog
```

Returns all worklogs for an issue, including:
- Worklog ID (needed for deletion)
- Author information (email, accountId)
- Started timestamp
- Time spent

### Delete Worklog

```
DELETE /ex/jira/{cloudId}/rest/api/3/issue/{issueIdOrKey}/worklog/{worklogId}
```

Returns 204 No Content on success.

## Implementation Flow

### Delete Mode Algorithm

When `--delete` or `-d` flag is used:

```
For each project/week combination in data file:
  1. Calculate Monday date for that week
  2. GET all worklogs for the ticket
  3. Filter worklogs by:
     - Author email matches JIRA_EMAIL
     - Started date matches calculated Monday (same day)
  4. For each matching worklog:
     - If dry-run: Display "would delete worklog {id}: {hours}h"
     - If not dry-run: DELETE the worklog
     - Report success/failure
```

### Safety Filters

**Author Matching:**
```go
func isMyWorklog(worklog Worklog, myEmail string) bool {
    return strings.EqualFold(worklog.Author.EmailAddress, myEmail)
}
```

**Date Matching:**
```go
func isSameDay(worklogStarted string, targetMonday time.Time) bool {
    worklogDate, err := time.Parse(time.RFC3339, worklogStarted)
    if err != nil {
        return false
    }

    wYear, wMonth, wDay := worklogDate.Date()
    tYear, tMonth, tDay := targetMonday.Date()

    return wYear == tYear && wMonth == tMonth && wDay == tDay
}
```

## Jira Client Extensions

Add to `internal/jira/client.go`:

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

func (c *Client) GetWorklogs(issueKey string) ([]Worklog, error) {
    url := fmt.Sprintf("%s/ex/jira/%s/rest/api/3/issue/%s/worklog",
        c.credentials.BaseURL,
        c.credentials.CloudID,
        issueKey,
    )

    // Make GET request
    // Parse response
    // Return worklogs
}

func (c *Client) DeleteWorklog(issueKey, worklogID string) error {
    url := fmt.Sprintf("%s/ex/jira/%s/rest/api/3/issue/%s/worklog/%s",
        c.credentials.BaseURL,
        c.credentials.CloudID,
        issueKey,
        worklogID,
    )

    // Make DELETE request
    // Check for 204 No Content
    // Return error if not successful
}

func (c *Client) FindMatchingWorklogs(issueKey string, targetDate time.Time, userEmail string) ([]Worklog, error) {
    worklogs, err := c.GetWorklogs(issueKey)
    if err != nil {
        return nil, err
    }

    matches := make([]Worklog, 0)
    for _, wl := range worklogs {
        if isMyWorklog(wl, userEmail) && isSameDay(wl.Started, targetDate) {
            matches = append(matches, wl)
        }
    }

    return matches, nil
}
```

## Command Line Interface

### Delete Flags

```bash
# Delete all hours for a month
jira-hours log --month 2026-01 --delete

# Short form
jira-hours log --month 2026-01 -d

# Delete specific week
jira-hours log --month 2026-01 --week 3 --delete

# Delete specific project
jira-hours log --month 2026-01 --code NPU --delete

# Dry run (show what would be deleted)
jira-hours log --month 2026-01 --delete --dry-run
```

### Delete Output

```
Deleting Hours: 2026-01
========================

Week 1 (Monday, 2026-01-05):
  NPU-25 (NPU):
    Found worklog 36643: 12h on 2026-01-05
    ✓ Deleted worklog 36643
  BCN-13815 (MGNT):
    Found worklog 36644: 24h on 2026-01-05
    Found worklog 36645: 24h on 2026-01-05  # Duplicate!
    ✓ Deleted worklog 36644
    ✓ Deleted worklog 36645
  BCN-17640 (SEC):
    No matching worklog found (skipped)

Week 2 (Monday, 2026-01-12):
  ...

Summary:
========
  Deleted: 17 worklogs
  Not found: 6 entries
  Failed: 0
  Total hours removed: 145
```

## Use Cases

### 1. Incorrect Hours

**Problem**: Logged 10h instead of 12h to NPU week 1

**Solution**:
```bash
# Delete the incorrect entry
./bin/jira-hours log --month 2026-01 --week 1 --code NPU --delete

# Update data/2026-01.yaml to correct hours
# Re-log with correct hours
./bin/jira-hours log --month 2026-01 --week 1 --code NPU
```

### 2. Wrong Week

**Problem**: Logged week 2 hours to week 3 by mistake

**Solution**:
```bash
# Delete week 3 (which has wrong data)
./bin/jira-hours log --month 2026-01 --week 3 --delete

# Update data file to move hours to correct week
# Re-log corrected data
./bin/jira-hours log --month 2026-01
```

### 3. Accidental Double-Run

**Problem**: Ran log command twice, created duplicates

**Solution**:
```bash
# With idempotent add (future): Just re-run, it will skip duplicates
./bin/jira-hours log --month 2026-01

# Without idempotent add: Delete all and re-log
./bin/jira-hours log --month 2026-01 --delete --dry-run  # Verify
./bin/jira-hours log --month 2026-01 --delete            # Execute
./bin/jira-hours log --month 2026-01                     # Re-log
```

### 4. Wrong Project Code

**Problem**: Logged NPU hours to MGNT by mistake in data file

**Solution**:
```bash
# Delete the incorrect MGNT entries
./bin/jira-hours log --month 2026-01 --code MGNT --delete

# Fix data file (move hours from MGNT to NPU)
# Re-log all (idempotent will skip correct entries)
./bin/jira-hours log --month 2026-01
```

## Integration with Idempotent Add

### Combined Workflow

With both features implemented:

1. **First time logging**: Adds all worklogs
2. **Discover error**: Some hours were wrong
3. **Fix data file**: Update YAML with correct hours
4. **Delete incorrect entries**: `--delete` flag removes old worklogs
5. **Re-log**: Idempotent add only adds what's missing

### Smart Sync Behavior

Future enhancement - detect differences:

```bash
jira-hours sync --month 2026-01
```

Would:
- GET all existing worklogs for the month
- Compare with data file
- Delete worklogs not in data file
- Add missing worklogs from data file
- Update worklogs with different hours

## Error Handling

### Multiple Worklogs Found

If multiple worklogs match the criteria (same date, same user):
- Delete all of them (they're likely duplicates)
- Report count: "Deleted 3 worklogs"

### No Worklog Found

If no matching worklog exists:
- Skip silently (nothing to delete)
- In verbose mode: Report "No worklog found for {ticket} on {date}"

### Partial Failures

If some deletions fail:
- Continue processing other entries
- Collect all errors
- Report summary at end with worklog IDs for manual cleanup

### Permission Errors

If user lacks permission to delete:
- Report clear error: "Cannot delete worklog {id}: permission denied"
- Suggest checking Jira permissions
- Provide worklog URL for manual deletion

## Testing Strategy

### Mock Delete Client

```go
type MockClient struct {
    worklogs map[string][]Worklog  // issueKey -> worklogs
}

func (m *MockClient) GetWorklogs(issueKey string) ([]Worklog, error) {
    return m.worklogs[issueKey], nil
}

func (m *MockClient) DeleteWorklog(issueKey, worklogID string) error {
    // Remove from mock storage
    return nil
}
```

### Test Scenarios

1. Delete existing worklog - should succeed
2. Delete non-existent worklog - should skip
3. Delete with multiple matches - should delete all
4. Delete other user's worklog - should skip (author filter)
5. Delete wrong date - should skip (date filter)

## API Rate Limiting

Since delete requires GET + DELETE for each entry:
- Each project/week needs 1 GET + N DELETEs
- More API calls than add mode
- Implement delays between requests
- Consider batch processing

## Summary Statistics

Track and report:
- `deleted`: Successfully deleted worklogs
- `notFound`: No matching worklog to delete
- `failed`: Deletion attempts that failed
- `totalHours`: Sum of hours from deleted worklogs

## Recommended Workflow

1. **Always dry-run first**:
   ```bash
   jira-hours log --month 2026-01 --delete --dry-run
   ```

2. **Review what will be deleted**

3. **Execute deletion**:
   ```bash
   jira-hours log --month 2026-01 --delete
   ```

4. **Verify in Jira** or check output for failures

5. **Re-log corrected hours**:
   ```bash
   jira-hours log --month 2026-01
   ```

This ensures you never accidentally delete the wrong worklogs.
