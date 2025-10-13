# Pre-commit Hooks Setup Summary

## Quick Start

### Installation (One-time setup)

```bash
# Option 1: Automated setup (recommended)
./scripts/setup-precommit.sh

# Option 2: Manual setup
pip install pre-commit
pre-commit install
pre-commit run --all-files  # Optional initial run
```

### Verification

After installation, the hooks will run automatically on `git commit`. You can test them:

```bash
# Run all hooks on staged files
pre-commit run

# Run all hooks on all files
pre-commit run --all-files

# Run a specific hook
pre-commit run golangci-lint --all-files
```

## What Gets Checked

### Stage: commit

- **File cleanup**: trailing whitespace, end-of-file, YAML syntax
- **Go quality**: fmt, imports, vet, build, golangci-lint
- **Security**: gosec, detect-secrets, forbidden patterns
- **Documentation**: markdownlint, yamllint
- **Docker**: hadolint (Dockerfile linting)
- **Protobuf**: buf lint, auto-generation
- **Shell scripts**: shellcheck
- **Custom**: go-test-changed (runs tests for modified packages)

### Stage: push (optional)

- **security-tests**: Full security test suite (manual trigger)

## Configuration Files

| File | Purpose | Status |
|------|---------|--------|
| `.pre-commit-config.yaml` | Main pre-commit configuration | ✅ Created |
| `.golangci.yml` | golangci-lint settings | ✅ Exists |
| `.yamllint.yml` | YAML linting rules | ✅ Created |
| `.markdownlint.json` | Markdown linting rules | ✅ Exists |
| `.secrets.baseline` | detect-secrets false positives | ✅ Created (empty) |

## Common Workflows

### Normal Development

```bash
# 1. Make changes
vim internal/gpio/handler.go

# 2. Stage changes
git add internal/gpio/handler.go

# 3. Commit (hooks run automatically)
git commit -m "feat: improve GPIO handler"

# Hooks run: file checks → Go checks → security → tests
# ✓ All hooks pass → commit succeeds
# ✗ Any hook fails → commit blocked, fix issues
```

### Skip Hooks (Use Sparingly)

```bash
# Skip all hooks (emergency only)
SKIP=pre-commit git commit -m "urgent fix"

# Skip specific hook
SKIP=golangci-lint git commit -m "wip: refactoring"

# Skip multiple hooks
SKIP=golangci-lint,gosec git commit -m "wip"
```

⚠️ **Warning**: Skipped hooks will still run in CI. Use this only for:

- Work-in-progress commits
- Quick local debugging
- Emergency hotfixes (CI must still pass)

### Update Hooks

```bash
# Update all hook repositories to latest versions
pre-commit autoupdate

# This updates versions in .pre-commit-config.yaml
```

## Hook Details

### golangci-lint

- **Runtime**: 10-30 seconds
- **Config**: `.golangci.yml`
- **What it checks**: 30+ linters (errcheck, gosimple, govet, staticcheck, etc.)
- **Bypass line**: `//nolint:linter-name // reason`

### gosec

- **Runtime**: 5-15 seconds
- **What it checks**: Security vulnerabilities (SQL injection, weak crypto, etc.)
- **Bypass line**: `// #nosec G101 // reason`

### detect-secrets

- **Runtime**: 2-5 seconds
- **Config**: `.secrets.baseline`
- **What it checks**: AWS keys, tokens, passwords, private keys
- **Update baseline**: `detect-secrets scan --baseline .secrets.baseline`

### go-test-changed (custom)

- **Runtime**: Varies (only runs tests for changed packages)
- **What it does**: Runs `go test` on modified packages
- **Example**: Change `internal/gpio/handler.go` → runs `go test ./internal/gpio/...`

### check-forbidden-patterns (custom)

- **Runtime**: <1 second
- **What it blocks**:
  - `fmt.Print*` (use structured logging)
  - `panic()` (use proper error handling)
  - `TODO SECURITY` (security todos must be resolved)

### proto-generated (custom)

- **Runtime**: 5-10 seconds
- **What it does**: Auto-generates protobuf files if `.proto` files changed
- **Command**: Runs `make proto`

## Troubleshooting

### "pre-commit: command not found"

```bash
# Install pre-commit
pip install pre-commit
# or
brew install pre-commit

# Then install hooks
pre-commit install
```

### "golangci-lint is slow"

- First run downloads tools (cached after)
- Subsequent runs: 10-30s
- To skip temporarily: `SKIP=golangci-lint git commit`

### "detect-secrets failing on legitimate files"

```bash
# Add false positives to baseline
detect-secrets scan --baseline .secrets.baseline

# Then commit the updated baseline
git add .secrets.baseline
git commit -m "chore: update secrets baseline"
```

### "Hook failed but I need to commit"

```bash
# Option 1: Fix the issue (recommended)
# Run the hook manually to see full output
pre-commit run golangci-lint

# Option 2: Skip hook (temporary)
SKIP=golangci-lint git commit -m "wip: will fix"

# Option 3: Disable all hooks (emergency)
git commit --no-verify -m "emergency fix"
```

⚠️ **Remember**: CI will still run all checks. Skipping locally just delays the feedback.

### "proto generation fails"

```bash
# Install protoc
brew install protobuf

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Run manually
make proto
```

## Performance Tips

### First Run

- **Expected**: 2-5 minutes (downloads all tools)
- **Downloads**: golangci-lint, gosec, detect-secrets, buf, shellcheck, etc.
- **One-time**: Tools are cached for future runs

### Subsequent Runs

- **Typical**: 15-45 seconds
- **Fast path**: Only checks staged files
- **Slow path**: `--all-files` checks everything (1-2 minutes)

### Optimization

```bash
# Only commit what you changed (faster)
git add specific-file.go

# vs

# Commit everything (slower if many files)
git add .
```

## Integration with IDE

### VS Code

Pre-commit hooks work seamlessly with VS Code's Git integration. Just commit through VS Code UI or terminal.

### GoLand/IntelliJ

Works with IDE's Git integration. Consider installing:

- Go fmt on save
- golangci-lint plugin
- detect-secrets plugin

## CI Comparison

| Check | Local (pre-commit) | CI (GitHub Actions) |
|-------|-------------------|---------------------|
| go fmt | ✅ <1s | ✅ 5s |
| go vet | ✅ 2s | ✅ 10s |
| golangci-lint | ✅ 15s | ✅ 30s |
| gosec | ✅ 10s | ✅ 20s |
| Unit tests | ✅ 5-30s (changed only) | ✅ 1-2m (all) |
| Security tests | ❌ Manual | ✅ 3-5m |
| Integration tests | ❌ Manual | ✅ 5-10m |

**Benefit**: Catch 80% of issues locally in 30-60s vs waiting 5-10 minutes for CI.

## Maintenance

### Monthly Tasks

```bash
# Update hook versions
pre-commit autoupdate

# Review and clean secrets baseline
detect-secrets audit .secrets.baseline

# Review golangci-lint configuration
# Check for new linters or deprecated settings
```

### When to Update

- New linter available
- Go version upgrade
- Security best practices change
- Team feedback on false positives

## Support

For issues with pre-commit hooks:

1. Check this document for troubleshooting
2. See `CONTRIBUTING.md` for detailed setup
3. Check `.pre-commit-config.yaml` for hook configuration
4. Open an issue if problem persists

## Additional Resources

- [pre-commit framework docs](https://pre-commit.com/)
- [golangci-lint documentation](https://golangci-lint.run/)
- [detect-secrets documentation](https://github.com/Yelp/detect-secrets)
- [buf documentation](https://buf.build/docs/)

---

**Last Updated**: 2025-01-13
**Maintained By**: Pi-Controller Team
