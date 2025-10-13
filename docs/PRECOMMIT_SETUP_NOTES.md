# Pre-commit Setup Notes

## Changes Made

### Issues Fixed

1. **Python Version Check** - Fixed version comparison in `scripts/setup-precommit.sh` that failed with Python 3.12 (was comparing "3.12" as decimal, now compares major/minor as integers)

2. **gosec Integration** - gosec v2.22.9 doesn't provide `.pre-commit-hooks.yaml`, so moved it to a local hook instead:

   ```yaml
   - id: gosec
     name: Security scan with gosec
     entry: bash -c 'gosec -quiet -exclude-generated ./...'
     language: system
     pass_filenames: false
     files: \.go$
   ```

3. **detect-secrets Removed** - detect-secrets v1.5.0 has pre-commit manifest issues. Removed from configuration since TruffleHog already handles secret detection in CI.

4. **Global stages removed** - The `stages:` configuration at root level caused warnings. Individual hooks can specify their own stages (like `security-tests` with `stages: [manual]`).

### Working Configuration

The `.pre-commit-config.yaml` now includes:

**External Repos:**

- `pre-commit/pre-commit-hooks` - File checks
- `dnephin/pre-commit-golang` - Go formatting/build
- `golangci/golangci-lint` - Comprehensive linting
- `igorshubovych/markdownlint-cli` - Markdown linting
- `adrienverge/yamllint` - YAML linting
- `hadolint/hadolint` - Dockerfile linting
- `bufbuild/buf` - Protobuf linting
- `shellcheck-py/shellcheck-py` - Shell script linting

**Local Hooks:**

- `gosec` - Security scanning (local hook, requires gosec installed)
- `go-test-changed` - Runs tests on modified packages
- `check-forbidden-patterns` - Blocks problematic code patterns
- `proto-generated` - Auto-generates protobuf files
- `security-tests` - Full security suite (manual trigger only)

## Installation

```bash
# Run the setup script
./scripts/setup-precommit.sh

# Or manual setup
pip install pre-commit
pre-commit install
```

## Prerequisites

For all hooks to work, you need:

- Python 3.8+
- Go 1.25.1+
- gosec (for local security scanning): `go install github.com/securego/gosec/v2/cmd/gosec@latest`
- protoc (for protobuf generation): `brew install protobuf`

## Usage

### Normal Workflow

```bash
git add .
git commit -m "feat: my changes"
# Hooks run automatically
```

### Run Manually

```bash
# All hooks on all files
pre-commit run --all-files

# Specific hook
pre-commit run golangci-lint

# Only on staged files
pre-commit run
```

### Skip Hooks (Use Sparingly)

```bash
# Skip specific hook
SKIP=golangci-lint git commit -m "wip"

# Skip all hooks (emergency only)
git commit --no-verify
```

## Performance

**First Run:** 2-5 minutes (downloads and caches tools)

**Subsequent Runs:**

- Small changes (1-3 files): 15-30s
- Medium changes (5-10 files): 30-60s
- Large changes: 1-2min

## Troubleshooting

### gosec not found

```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

### Hooks are slow

- First run downloads tools (one-time)
- Use `SKIP=golangci-lint` for quick iterations
- CI will still check everything

### Hook failed but need to commit

```bash
# View full output
pre-commit run <hook-name>

# Skip temporarily (not recommended)
SKIP=<hook-name> git commit

# Emergency bypass (use with caution)
git commit --no-verify
```

## Future Improvements

1. **Consider re-adding detect-secrets** when a stable pre-commit integration is available
2. **Add custom hooks** for project-specific checks
3. **Tune golangci-lint timeout** if needed for large changesets
4. **Add pre-push hooks** for longer-running checks

## References

- Pre-commit framework: <https://pre-commit.com/>
- Configuration file: `.pre-commit-config.yaml`
- Setup script: `scripts/setup-precommit.sh`
- Full documentation: `docs/PRECOMMIT_HOOKS.md`
- Quick reference: `docs/PRECOMMIT_QUICKREF.md`

---
**Last Updated:** October 13, 2025
