# Pre-Commit and CI Parity Guide

## Philosophy: Develop Locally, Deploy Confidently

**The Golden Rule**: If it passes locally, it MUST pass in CI.

Pre-commit hooks exist to catch issues **before** they reach the pipeline. There's no point in running local checks if they don't match what CI does - it wastes time, creates frustration, and defeats the entire purpose of local validation.

## Configuration Parity Checklist

### Security Scanning (gosec)

**✅ Current Configuration (CORRECT)**

- **Local**: `gosec -exclude-generated ./...`
- **CI**: `gosec -exclude-generated -fmt sarif ./...`
- **Match**: Both exclude generated files

**❌ Previous Problem (INCORRECT)**

- **Local**: `gosec -quiet -exclude-generated ./...` (quiet mode, excludes generated)
- **CI**: `gosec -fmt sarif ./...` (no exclusions)
- **Mismatch**: CI caught 9 issues in generated files that local didn't

### Linting (golangci-lint)

**✅ Configuration**

- **Local**: `golangci-lint run --timeout=5m --config=.golangci.yml`
- **CI**: `golangci-lint run --out-format sarif`
- **Match**: Both use same `.golangci.yml` config

### Format Checking

**✅ Configuration**

- **Local**: `go fmt`, `go imports`
- **CI**: `make deps` (includes go fmt, go imports)
- **Match**: Same tools run

## How to Maintain Parity

### 1. When Adding New CI Checks

**WRONG WAY** ❌

```yaml
# CI only - local hooks don't match
- name: Run new security check
  run: super-secure-tool --strict ./...
```

**RIGHT WAY** ✅

```yaml
# CI workflow
- name: Run new security check
  run: super-secure-tool --strict ./...
```

```yaml
# .pre-commit-config.yaml
- repo: local
  hooks:
    - id: super-secure-tool
      name: Security check with super-secure-tool
      entry: bash -c 'super-secure-tool --strict ./...'
      language: system
```

### 2. When Updating Tool Flags

**If you change CI flags, update pre-commit hooks immediately.**

Example: Adding `-exclude-generated` to gosec

1. Update `.github/workflows/security.yml`
2. Update `.pre-commit-config.yaml`
3. Test locally: `pre-commit run --all-files`
4. Commit both changes together

### 3. Testing Parity

**Before pushing any hook/CI changes:**

```bash
# Test all hooks locally
pre-commit run --all-files

# If all pass, push and immediately check CI
git push origin develop

# Watch the workflow run
gh run watch

# If CI fails but local passed = PARITY BROKEN
# Fix immediately, don't move forward
```

## Common Parity Issues

### Issue: Generated Files

**Problem**: CI scans generated code, local doesn't (or vice versa)

**Solution**:

- Always use `-exclude-generated` in both places
- Add explicit exclusions in `.golangci.yml` if needed
- Document what's considered "generated" (`.pb.go`, `mock_*.go`, etc.)

### Issue: Different Tool Versions

**Problem**: Local uses go 1.24, CI uses go 1.25

**Solution**:

- Pin Go version in `.pre-commit-config.yaml`
- Use same version specified in CI workflow
- Use `asdf` or similar to manage local Go version

### Issue: Quiet/Silent Modes

**Problem**: Local hook uses `-quiet` flag, hides real issues

**Solution**:

- NEVER use quiet/silent modes in pre-commit hooks
- You WANT to see failures locally
- Only use output formatting differences (not suppression)

### Issue: Conditional Checks

**Problem**: CI has `if: github.event_name == 'pull_request'` but local always runs

**Solution**:

- Make local checks the SUPERSET of CI checks
- Better to run extra checks locally than miss CI failures
- Use `stages: [manual]` for expensive optional checks

## Debugging Parity Failures

When CI fails but local passed:

1. **Check exact command differences**

   ```bash
   # Local
   grep -A 5 "id: gosec" .pre-commit-config.yaml

   # CI
   grep -A 10 "Run gosec" .github/workflows/security.yml
   ```

2. **Look for flag differences**
   - Exclusion flags: `-exclude`, `-skip`, `--ignore`
   - Format flags: `-fmt`, `--format`
   - Verbosity: `-quiet`, `-verbose`, `--silent`

3. **Check file targeting**

   ```bash
   # Local might target specific files
   files: \.go$

   # CI might target everything
   run: tool ./...
   ```

4. **Reproduce CI environment locally**

   ```bash
   # Use CI's exact command
   gosec -exclude-generated -fmt json -out report.json ./...

   # Compare to pre-commit hook command
   ```

## Enforcement

### Pull Request Requirements

- [ ] All pre-commit hooks pass locally
- [ ] CI security workflow passes
- [ ] If hook/CI configs changed, both are updated
- [ ] No `-quiet` or `-silent` flags in pre-commit hooks
- [ ] Tool versions match between local and CI

### Code Review Checklist

When reviewing PRs that touch `.pre-commit-config.yaml` or `.github/workflows/`:

- [ ] Check flags match between local and CI
- [ ] Verify exclusions are consistent
- [ ] Ensure no silent/quiet modes added
- [ ] Confirm same tool versions
- [ ] Test changes locally with `pre-commit run --all-files`

## Quick Reference

### Testing Local Hooks

```bash
# Run all hooks
pre-commit run --all-files

# Run specific hook
pre-commit run gosec --all-files

# Run on specific files
pre-commit run --files internal/services/*.go

# Show what would run (dry run)
pre-commit run --all-files --verbose --show-diff-on-failure
```

### Checking CI Status

```bash
# View latest workflow runs
gh run list --workflow=security.yml --limit 5

# Watch current run
gh run watch

# View failed run logs
gh run view <run-id> --log-failed

# View specific job
gh run view <run-id> --log --job=static-analysis
```

### Common Fix Patterns

```bash
# Sync hooks with CI after workflow update
git add .github/workflows/security.yml .pre-commit-config.yaml
git commit -m "fix(ci): ensure gosec parity between local and CI"

# Test before pushing
pre-commit run --all-files && git push
```

## Maintenance

**Monthly Review**: First Monday of each month

- [ ] Compare all tool flags in `.pre-commit-config.yaml` vs CI workflows
- [ ] Check for new CI checks that lack local hooks
- [ ] Verify tool versions match (Go, golangci-lint, gosec)
- [ ] Test full pipeline: `pre-commit run --all-files`
- [ ] Document any intentional differences

## Summary

**Your local environment should be your safety net, not a false sense of security.**

Every minute spent fixing parity issues saves hours of CI debugging and developer frustration. When in doubt, make local checks STRICTER than CI, never weaker.

---

**Last Updated**: 2025-10-13
**Maintained By**: DevOps/Security Team
**Related Docs**:

- [PRECOMMIT_HOOKS.md](./PRECOMMIT_HOOKS.md) - Hook installation and usage
- [CI_STRATEGY.md](../CI_STRATEGY.md) - Overall CI/CD approach
- [SECURITY_IMPLEMENTATION.md](./SECURITY_IMPLEMENTATION.md) - Security tooling details
