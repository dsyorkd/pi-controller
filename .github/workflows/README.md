# GitHub Actions Workflows

This directory contains the modular CI/CD workflows for the pi-controller project

## 🏗️ Architecture Overview

The workflow structure is designed to be **lightweight**, **modular**, and **efficient**:

```
ci.yml (Orchestrator)
├── validate.yml       # Lint, format, vet checks
├── test.yml           # Unit, integration, security tests
├── build.yml          # Multi-arch builds
└── hardware-test      # Hardware testing (inline)

performance.yml        # Heavy benchmarks (master only)
```

## 📋 Workflow Descriptions

### Primary Workflows

#### `ci.yml` - Main CI Orchestrator

**Trigger**: Every push/PR to `main`, `develop`, `master`

- Determines CI strategy based on branch
- Calls modular workflows (`validate.yml`, `test.yml`, `build.yml`)
- Runs hardware tests conditionally
- Provides unified status check

**Duration**: ~6-8 minutes for typical PR

#### `validate.yml` - Code Validation

**Trigger**: Called by `ci.yml` or can run independently

- Go formatting check (`go fmt`)
- Go vet analysis
- golangci-lint
- Protobuf code generation

**Duration**: ~3-5 minutes

#### `test.yml` - Comprehensive Testing

**Trigger**: Called by `ci.yml` or can run independently

**Jobs**:

1. **Unit & Integration Tests**: Core functionality testing
2. **Race Condition Tests**: Data race detection with `-race` flag
3. **Database Migrations**: Migration up/down cycle testing

**Duration**: ~5-6 minutes

#### `build.yml` - Multi-Architecture Builds

**Trigger**: Called by `ci.yml` or can run independently

Builds binaries for:

- `linux-amd64`
- `linux-arm64`
- `linux-arm`
- `darwin-amd64`
- `darwin-arm64`

**Duration**: ~4-6 minutes (parallel builds)

#### `performance.yml` - Performance & Benchmarks

**Trigger**:

- ✅ Pushes to `main`/`master` only
- ✅ Manual dispatch (`workflow_dispatch`)
- ✅ Weekly schedule (Sunday 2 AM UTC)

**Jobs**:

1. **Benchmarks**: GPIO, services, comprehensive benchmarks
2. **Fuzzing**: 8-minute fuzz testing session
3. **Stress Testing**: Comprehensive test suite

**Duration**: ~15 minutes
**Note**: This is the **only** heavy workflow - kept off develop/PR branches

### Specialized Workflows

#### `hardware-testing.yml`

**Trigger**: Manual or specific events

Runs on self-hosted Raspberry Pi hardware

#### `security.yml`, `scan.yml`, `codeqa-analysis.yml`

Security scanning and code quality analysis

#### `build-multiarch.yml`

Docker multi-architecture builds

#### `release.yml`

Release automation when tags are pushed

## 🎯 Branch Strategy

### `develop` Branch (Lightweight)

✅ Runs: `validate.yml`, `test.yml`, `build.yml`
❌ Skips: `performance.yml`, hardware tests (unless labeled)

**Total Duration**: ~8-10 minutes

### `main`/`master` Branch (Comprehensive)

✅ Runs: All workflows including `performance.yml`
✅ Includes: Hardware tests (if available)

**Total Duration**: ~20-25 minutes (with performance tests running separately)

### Pull Requests

✅ Runs: `validate.yml`, `test.yml`, `build.yml`
❌ Skips: Hardware tests (unless PR has `hw-test` label)

**Total Duration**: ~8-10 minutes

## 🚀 Performance Optimization

### Before Restructuring

- Single monolithic CI file
- All tests run on every push
- Performance tests on every PR (~30 minutes total)

### After Restructuring

- Modular, reusable workflows
- Performance tests only on master
- **70% faster** for typical develop/PR workflow
- Parallel execution where possible

## 🔧 Usage Examples

### Running Specific Workflows Manually

```bash
# Trigger validation only
gh workflow run validate.yml

# Trigger performance tests manually (if needed on feature branch)
gh workflow run performance.yml

# Trigger build for specific branch
gh workflow run build.yml --ref your-branch-name
```

### Adding Hardware Tests to a PR

Add the `hw-test` label to your PR:

```bash
gh pr edit <PR-NUMBER> --add-label hw-test
```

### Checking Workflow Status

```bash
# View recent workflow runs
gh run list

# View specific workflow
gh run list --workflow=ci.yml

# Watch a specific run
gh run watch <RUN-ID>
```

## 📊 Workflow Matrix

| Workflow | develop | main/master | PR | Manual |
|----------|---------|-------------|-----|--------|
| `validate.yml` | ✅ | ✅ | ✅ | ✅ |
| `test.yml` | ✅ | ✅ | ✅ | ✅ |
| `build.yml` | ✅ | ✅ | ✅ | ✅ |
| `performance.yml` | ❌ | ✅ | ❌ | ✅ |
| Hardware Test | ❌ | ✅ | w/ label | ✅ |

## 🎨 Adding a New Workflow

1. Create workflow file in `.github/workflows/`
2. Add `workflow_call:` trigger to make it reusable:

   ```yaml
   on:
     workflow_call:
   ```

3. Call it from `ci.yml`:

   ```yaml
   my-new-job:
     uses: ./.github/workflows/my-workflow.yml
     needs: [validate]
   ```

## 🐛 Troubleshooting

### Workflow not triggering?

- Check branch name matches trigger conditions
- Verify `workflow_call` is present for reusable workflows
- Check workflow syntax with: `gh workflow view <workflow-name>`

### Performance tests not running?

- Performance tests only run on `main`/`master` pushes
- Manually trigger with: `gh workflow run performance.yml`

### Build artifacts missing?

- Check artifact retention period (7-30 days)
- Download with: `gh run download <RUN-ID>`

## 📚 Additional Resources

- [GitHub Actions Reusable Workflows](https://docs.github.com/en/actions/using-workflows/reusing-workflows)
- [Project Makefile](../../Makefile) - Understanding test targets
