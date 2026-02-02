# Code Validation Skill

This skill validates that code changes pass all quality checks before marking tasks as complete.

## Trigger Phrases

Use this skill when:

- About to mark a coding task as complete
- User asks to validate code changes
- Need to verify pre-commit hooks pass
- Need to verify builds succeed
- Need to run tests on changed code

## Validation Steps

When validating code, perform these checks in order:

### 1. Build Verification

```bash
go build ./...
```

- If build fails, report the errors and fix them before proceeding
- Do NOT mark task as complete if build fails

### 2. Pre-commit Hooks

```bash
pre-commit run --all-files
```

- Check for:
  - Go formatting issues (gofmt, goimports)
  - Linting issues (golangci-lint)
  - Markdown formatting
  - Trailing whitespace
  - YAML/JSON validation

### 3. Unit Tests on Changed Packages

```bash
go test -short -race ./...
```

- Run tests with race detection
- Report any test failures

### 4. Security Tests (if security-related changes)

```bash
make test-security
```

## Validation Report Format

After running validation, report results in this format:

```
## Validation Results

| Check | Status | Details |
|-------|--------|---------|
| Build | PASS/FAIL | Error details if any |
| Pre-commit | PASS/FAIL | Failed hooks if any |
| Tests | PASS/FAIL | Failed tests if any |

### Issues Found (if any)
- Issue 1: description
- Issue 2: description

### Recommended Actions
1. Action to fix issue 1
2. Action to fix issue 2
```

## Important Rules

1. **NEVER mark a task as complete if validation fails**
2. Fix all issues before completing the task
3. Re-run validation after fixes
4. Document any issues that cannot be fixed and require follow-up

## Common Issues and Fixes

### Duplicate Imports

- Remove duplicate import statements
- Run `goimports -w <file>` to auto-fix

### Missing Arguments

- Check function signatures have changed
- Update all call sites

### Forbidden Patterns

- `fmt.Print*` - Use logger instead (except in cmd/ main files)
- `panic()` - Use error returns instead
- `TODO SECURITY` / `FIXME SECURITY` - Address before merge

### Markdown Issues

- Multiple H1 headers - Use single # header per file
- Trailing whitespace - Auto-fixed by pre-commit
