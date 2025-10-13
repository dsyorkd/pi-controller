# Pre-commit Hooks - Quick Reference Card

## Setup (One-time)

```bash
./scripts/setup-precommit.sh
```

## Daily Use

### Normal Commit

```bash
git add .
git commit -m "feat: my changes"
# Hooks run automatically ✓
```

### Skip Hook (Temporary)

```bash
SKIP=golangci-lint git commit -m "wip"
```

### Manual Run

```bash
pre-commit run --all-files
```

## Common Commands

| Task | Command |
|------|---------|
| Run all hooks | `pre-commit run --all-files` |
| Run specific hook | `pre-commit run golangci-lint` |
| Skip all hooks | `SKIP=pre-commit git commit` |
| Skip one hook | `SKIP=gosec git commit` |
| Update hooks | `pre-commit autoupdate` |
| Uninstall | `pre-commit uninstall` |

## Installed Hooks

### Fast (<5s)

- ✓ File checks (whitespace, EOF, YAML)
- ✓ Go fmt, imports, vet
- ✓ detect-secrets
- ✓ Forbidden patterns

### Medium (5-30s)

- ✓ Go build
- ✓ golangci-lint
- ✓ gosec
- ✓ go-test-changed

### Slow (30s+)

- ⏱ Protobuf generation (if .proto changed)

## Troubleshooting

### Hook failed?

```bash
# See full output
pre-commit run <hook-name>

# Fix issues, then retry
git add .
git commit -m "fix: issues"
```

### Need to commit NOW?

```bash
git commit --no-verify
# ⚠️ CI will still check!
```

### Update secrets baseline

```bash
detect-secrets scan --baseline .secrets.baseline
```

## Performance

| Scenario | Time |
|----------|------|
| First run (downloads) | 2-5 min |
| Normal commit | 15-45s |
| Changed 1 file | 10-20s |
| Changed 10+ files | 30-60s |

## Tips

💡 **Commit small changes frequently** → Faster hooks

💡 **Run `pre-commit run` before committing** → See issues early

💡 **Use `SKIP=` sparingly** → CI will catch issues anyway

💡 **Keep hooks updated** → `pre-commit autoupdate` monthly

## Help

📖 Full docs: `docs/PRECOMMIT_HOOKS.md`
📖 Contributing: `CONTRIBUTING.md`
🐛 Issues: Check troubleshooting section

---
*Last updated: 2025-01-13*
