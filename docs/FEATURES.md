# jira-hours Features

## Two Complementary Features for Error-Free Time Logging

### 1. Idempotent Add (Check Before Logging)

**Problem**: Running the log command twice creates duplicate worklogs

**Solution**: Before adding a worklog, check if it already exists

**How it works**:
1. Calculate Monday date for the week
2. GET existing worklogs for the ticket
3. Check if worklog exists with:
   - Same author (your email)
   - Same date (the Monday)
   - Same hours
4. If exists: Skip (report "already logged")
5. If not exists: Add the worklog

**Benefits**:
- Safe to re-run log command
- Can log partial month, then complete it later
- No duplicate entries

**Example**:
```bash
# First run
$ ./bin/jira-hours log --month 2026-01
✓ NPU-25 (NPU): 12h logged (new)

# Second run (idempotent)
$ ./bin/jira-hours log --month 2026-01
○ NPU-25 (NPU): 12h already logged (worklog 36643)
```

### 2. Delete Mode (Error Correction)

**Problem**: Logged incorrect hours and need to remove them

**Solution**: Delete mode finds and removes your worklogs

**How it works**:
1. Calculate Monday date for the week
2. GET existing worklogs for the ticket
3. Find worklogs with:
   - Same author (your email)
   - Same date (the Monday)
4. DELETE each matching worklog
5. Report what was deleted

**Benefits**:
- Correct mistakes without manual Jira cleanup
- Safe (only deletes your worklogs)
- Supports dry-run to preview deletions

**Example**:
```bash
# Preview what would be deleted
$ ./bin/jira-hours log --month 2026-01 --delete --dry-run
Would delete worklog 36643: NPU-25, 12h on 2026-01-05
Would delete worklog 36644: BCN-13815, 24h on 2026-01-05

# Actually delete
$ ./bin/jira-hours log --month 2026-01 --delete
✓ Deleted worklog 36643
✓ Deleted worklog 36644
```

## Combined Workflow: Error Correction

### Scenario: Logged Wrong Hours

You logged hours but made mistakes in the data file.

**Steps**:

1. **Discover the error**
   ```bash
   # Check what's in Jira (future: pull command)
   ```

2. **Fix your data file**
   ```bash
   # Edit data/2026-01.yaml with correct hours
   ```

3. **Delete incorrect entries**
   ```bash
   # Preview deletion
   ./bin/jira-hours log --month 2026-01 --delete --dry-run

   # Execute deletion
   ./bin/jira-hours log --month 2026-01 --delete
   ```

4. **Re-log with correct data**
   ```bash
   # Idempotent add will only log what's needed
   ./bin/jira-hours log --month 2026-01
   ```

## Selective Operations

Both add and delete support filters:

### By Week
```bash
# Add/delete only week 3
./bin/jira-hours log --month 2026-01 --week 3
./bin/jira-hours log --month 2026-01 --week 3 --delete
```

### By Project
```bash
# Add/delete only NPU entries
./bin/jira-hours log --month 2026-01 --code NPU
./bin/jira-hours log --month 2026-01 --code NPU --delete
```

### By Week AND Project
```bash
# Add/delete NPU week 3 only
./bin/jira-hours log --month 2026-01 --week 3 --code NPU
./bin/jira-hours log --month 2026-01 --week 3 --code NPU --delete
```

## Safety Features

### 1. Author Filtering
- Only operates on YOUR worklogs
- Never touches other users' entries
- Verifies email matches `JIRA_EMAIL`

### 2. Date Matching
- Matches by date only (ignores time)
- Handles timezone differences
- Considers same calendar day

### 3. Dry-Run Mode
- Always available with `--dry-run`
- Shows exactly what would happen
- No API modifications

### 4. Clear Reporting
- Shows worklog IDs for traceability
- Reports counts: added, deleted, skipped
- Provides total hours affected

## API Calls Comparison

### Add Mode (Without Idempotent Check)
- 1 POST per non-zero entry
- ~23 API calls for typical month

### Add Mode (With Idempotent Check)
- 1 GET per ticket (to check existing worklogs)
- 1 POST per new entry
- ~12 GETs + ~23 POSTs = ~35 API calls

### Delete Mode
- 1 GET per ticket (to find worklogs)
- 1 DELETE per matching worklog
- ~12 GETs + ~23 DELETEs = ~35 API calls

## Future Enhancements

### 1. Pull Command
```bash
# Download existing worklogs from Jira
jira-hours pull --month 2026-01 --output data/2026-01-jira.yaml
```

### 2. Diff Command
```bash
# Compare local file vs Jira
jira-hours diff --month 2026-01
```

### 3. Sync Command
```bash
# Make Jira match local file exactly
jira-hours sync --month 2026-01
```

### 4. Update Instead of Delete+Add
```bash
# Update worklog hours in place
PUT /worklog/{worklogId}
```

## Implementation Status

### ✓ Current (Implemented)
- Add worklog
- Dry-run mode
- Mock client for testing
- Validation
- Filters (week, code)

### ⧗ Planned (Designed, Not Implemented)
- Idempotent add (check before logging)
- Delete mode
- Get worklogs
- Pull command
- Sync command

### ○ Future Considerations
- Update worklog in place
- Batch operations
- Progress indicators
- Rollback capability
