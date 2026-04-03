# Changes Made - Environment Variables Only

## Summary

Removed all support for `configs/credentials.yaml`. The tool now **requires** environment variables for authentication.

## Files Changed

### Code Changes

1. **internal/config/config.go**
   - Removed `LoadCredentials()` function
   - Removed `CredentialsConfig` struct
   - Kept only `LoadCredentialsFromEnv()`
   - Updated error messages to mention environment variables

2. **cmd/hours/log.go**
   - Removed fallback to `configs/credentials.yaml`
   - Updated error message to reference `.envrc` or `.env`

3. **Makefile**
   - Renamed binary from `hours` to `jira-hours`
   - Updated all references

4. **All Go source files**
   - Updated module name from `github.com/gherlein/hours` to `github.com/gherlein/jira-hours`

### Documentation Changes

1. **README.md**
   - Removed Option B (credentials.yaml)
   - Updated all binary references to `jira-hours`
   - Clarified only environment variables supported

2. **QUICKSTART.md**
   - Removed credentials.yaml mentions
   - Added note about environment variables only
   - Updated binary name

3. **TESTING.md**
   - Removed credentials.yaml references
   - Updated binary name
   - Added note about environment-only auth

4. **docs/DESIGN.md**
   - Updated file structure (removed credentials.yaml)
   - Updated credentials section to show .envrc/.env
   - Removed --config flag from CLI flags
   - Updated security considerations
   - Updated config precedence section

5. **.envrc.example**
   - Updated with clear instructions
   - Shows all required variables

### File Deletions

1. **configs/credentials.yaml.example** - Deleted

### Configuration Changes

1. **.gitignore**
   - Removed `configs/credentials.yaml` (no longer needed)
   - Kept `.envrc`, `.env`, `.env.local`

## Authentication Requirements

The tool now **only** supports environment variables:

```bash
# Required
export JIRA_CLOUD_ID="29ad0f88-9969-4673-b232-4aa64e95f11b"
export JIRA_EMAIL="your-email@brightsign.biz"
export JIRA_API_TOKEN="your-api-token"

# Optional (defaults to https://api.atlassian.com)
export JIRA_BASE_URL="https://api.atlassian.com"
```

Set these in `.envrc` (with direnv) or `.env` file.

## Error Messages Updated

Old error:
```
credentials not found in environment or configs/credentials.yaml
```

New error:
```
credentials not found in environment variables (.envrc or .env): JIRA_CLOUD_ID environment variable required
```

## Testing Confirmed

All tests pass with the new environment-only authentication:

```bash
$ make run-test
✓ NPU-25 (NPU): 1h logged
✓ BCN-13815 (MGNT): 1h logged
Success: 2
Total hours: 2
```

## Why This Change?

1. **Simpler** - One authentication method, not two
2. **Safer** - Less chance of committing credentials
3. **Consistent** - Matches existing .envrc workflow
4. **Standard** - Environment variables are the Go standard for config
