# Implementation Plan: Delete and Idempotent Add Features

## Features Designed (Not Yet Implemented)

Two complementary features to make time logging error-proof:

### 1. Idempotent Add
Check if worklog already exists before adding.

### 2. Delete Mode
Remove incorrectly logged worklogs.

## API Endpoints Needed

### Already Used
- ✓ POST `/worklog` - Add worklog (implemented)

### Need to Add
- GET `/issue/{key}/worklog` - Get all worklogs for an issue
- DELETE `/issue/{key}/worklog/{id}` - Delete specific worklog

## Implementation Order

### Phase 1: Add Delete Mode
1. Add `GetWorklogs()` to jira client
2. Add `DeleteWorklog()` to jira client
3. Add helper functions for filtering worklogs
4. Add `--delete` flag to log command
5. Implement delete logic in log command
6. Update statistics tracking (deleted, notFound)
7. Test with mock client

### Phase 2: Add Idempotent Check
1. Add `WorklogExists()` to jira client (uses GetWorklogs)
2. Modify log command to check before adding
3. Add statistics for "already logged"
4. Update output formatting
5. Test idempotent behavior

### Phase 3: Combined Testing
1. Test delete mode alone
2. Test idempotent add alone
3. Test delete + re-add workflow
4. Test with real Jira API
5. Document edge cases found

## Code Changes Required

### 1. internal/jira/client.go

Add new types:
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

Add new methods:
```go
func (c *Client) GetWorklogs(issueKey string) ([]Worklog, error)
func (c *Client) DeleteWorklog(issueKey, worklogID string) error
func (c *Client) WorklogExists(issueKey string, targetDate time.Time, hours int, userEmail string) (bool, string, error)
func (c *Client) FindMatchingWorklogs(issueKey string, targetDate time.Time, userEmail string) ([]Worklog, error)
```

### 2. cmd/hours/log.go

Add flag:
```go
var deleteMode bool
cmd.Flags().BoolVarP(&deleteMode, "delete", "d", false, "Delete mode")
```

Update logHours function:
```go
func logHours(..., deleteMode bool, ...) error {
    // ... existing code ...

    if deleteMode {
        return deleteHours(monthlyLog, projectConfig, client, ...)
    }

    // Add mode with idempotent check
    return addHours(monthlyLog, projectConfig, client, ...)
}
```

Add new functions:
```go
func addHours(...) error {
    // For each entry:
    //   Check if exists with WorklogExists()
    //   Skip if exists, add if not
}

func deleteHours(...) error {
    // For each entry:
    //   Find matching worklogs with FindMatchingWorklogs()
    //   Delete each match with DeleteWorklog()
}
```

### 3. internal/jira/client_test.go (New File)

Add tests:
```go
func TestGetWorklogs(t *testing.T)
func TestDeleteWorklog(t *testing.T)
func TestWorklogExists(t *testing.T)
func TestFindMatchingWorklogs(t *testing.T)
func TestIdempotentAdd(t *testing.T)
func TestDeleteMode(t *testing.T)
```

### 4. MockClient Updates

Add to MockClient:
```go
type MockClient struct {
    loggedEntries []WorklogRequest
    existingWorklogs map[string][]Worklog  // issueKey -> worklogs
}

func (m *MockClient) GetWorklogs(issueKey string) ([]Worklog, error)
func (m *MockClient) DeleteWorklog(issueKey, worklogID string) error
```

## Testing Plan

### Unit Tests

1. **Date matching logic**
   - Same day, different time → match
   - Different day → no match
   - Timezone handling

2. **Author matching**
   - Same email → match
   - Different email → no match
   - Case insensitive comparison

3. **Worklog filtering**
   - Multiple worklogs on same day
   - Worklogs from different authors
   - Edge cases (no worklogs, all match, none match)

### Integration Tests

1. **Idempotent add**
   - First run: adds worklogs
   - Second run: skips (already logged)
   - Partial run: adds only missing

2. **Delete mode**
   - Delete existing: succeeds
   - Delete non-existent: skips
   - Delete with filters: only deletes matching

3. **Combined workflow**
   - Add → Delete → Re-add
   - Verify no duplicates
   - Verify correct final state

### Manual Testing with Mock

1. Create test data with known worklogs
2. Run add (should skip existing)
3. Run delete (should remove matches)
4. Run add again (should re-add)
5. Verify statistics correct

## Estimated Effort

### Phase 1: Delete Mode
- Client methods: 4 hours
- Command integration: 2 hours
- Testing: 2 hours
- **Total: 8 hours**

### Phase 2: Idempotent Add
- Client methods: 2 hours
- Command integration: 2 hours
- Testing: 2 hours
- **Total: 6 hours**

### Phase 3: Integration
- Combined testing: 2 hours
- Documentation: 1 hour
- Real Jira testing: 1 hour
- **Total: 4 hours**

**Grand Total: ~18 hours**

## Risk Mitigation

### Risk 1: Deleting Wrong Worklogs
**Mitigation**:
- Always filter by author email
- Require explicit --delete flag
- Mandatory dry-run recommendation in docs
- Show worklog IDs before deleting

### Risk 2: API Rate Limiting
**Mitigation**:
- Add delays between requests (100ms)
- Implement exponential backoff
- Batch operations when possible
- Cache worklog responses

### Risk 3: Timezone Issues
**Mitigation**:
- Match by date only (ignore time)
- Document timezone behavior
- Use consistent timezone (America/Los_Angeles)
- Handle ±12 hour window

### Risk 4: Incomplete Operations
**Mitigation**:
- Continue on errors (don't stop mid-operation)
- Report all failures at end
- Provide worklog IDs for manual cleanup
- Log state for resume capability

## Rollout Strategy

### Step 1: Delete Mode Only
- Implement and test delete functionality
- Release for error correction
- Gather feedback

### Step 2: Idempotent Add
- Implement check-before-add
- Test with existing delete mode
- Release combined features

### Step 3: Advanced Features
- Pull command (download from Jira)
- Sync command (make Jira match local)
- Diff command (compare)

## Success Criteria

### Delete Mode
- ✓ Can delete specific week/project
- ✓ Only deletes authenticated user's worklogs
- ✓ Dry-run shows accurate preview
- ✓ Reports what was deleted
- ✓ Handles "not found" gracefully

### Idempotent Add
- ✓ Detects existing identical worklogs
- ✓ Skips adding duplicates
- ✓ Reports "already logged" vs "new"
- ✓ Can re-run safely
- ✓ Adds only what's missing

### Combined
- ✓ Delete + re-add workflow works
- ✓ No duplicate entries created
- ✓ Correct final state in Jira
- ✓ Clear user feedback throughout
