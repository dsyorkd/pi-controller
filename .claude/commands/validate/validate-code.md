---
description: Validate code changes pass build, pre-commit hooks, and tests
---

# Code Validation

Run comprehensive validation on code changes before marking tasks complete.

## Instructions

You MUST run these validation steps in order and report results:

### Step 1: Build Verification

Run the Go build:

```bash
go build ./...
```

If the build fails:

1. Report the exact error messages
2. Fix the issues
3. Re-run the build until it passes

### Step 2: Pre-commit Hooks

Run pre-commit on all files:

```bash
pre-commit run --all-files
```

Check for failures in:

- `go-fmt` - Go formatting
- `go-imports` - Import organization
- `golangci-lint` - Linting
- `markdownlint` - Markdown formatting
- `go-test-changed` - Tests on changed packages
- `check-forbidden-patterns` - Disallowed code patterns

### Step 3: Unit Tests

Run tests with race detection:

```bash
go test -short -race ./... 2>&1 | tail -50
```

### Step 4: Report Results

Create a validation report:

```
## Validation Results

| Check | Status |
|-------|--------|
| Build | PASS/FAIL |
| Pre-commit | PASS/FAIL |
| Tests | PASS/FAIL |

### Issues Found
[List any issues]

### Status
- [ ] Ready to commit
- [ ] Needs fixes (list what)
```

## Important

- **DO NOT skip any validation step**
- **DO NOT mark tasks complete if validation fails**
- Fix all issues before declaring completion
- If issues cannot be fixed, document them clearly

## Common Fixes

| Issue | Fix |
|-------|-----|
| Duplicate imports | Remove duplicates, run `goimports -w file.go` |
| Build errors | Check error message, fix syntax/type issues |
| Lint failures | Follow golangci-lint suggestions |
| Test failures | Debug and fix failing tests |
| Forbidden patterns | Replace `fmt.Print*` with logger calls |
