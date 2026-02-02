# Pi-Controller Coding Standards

## Git Workflow
- **Working branch:** `develop`
- **Feature branches:** `feature/<task-id>-short-description` off develop
- **Commits:** Conventional Commits — `<type>(<scope>): <subject>`
- **Types:** feat, fix, docs, style, refactor, test, chore, build, ci
- **Git user:** billy-bot-ctrl / billy-bot-ctrl@users.noreply.github.com

## Go Standards
- All code formatted with `gofmt` — run `make fmt`
- Must pass `golangci-lint` — run `make lint`
- Must pass `go vet` — run `make vet`
- Errors handled explicitly — never discard with `_`
- Use `errors` package for wrapping/context
- Use sentinel errors (e.g., `ErrNotFound`, `ErrConflict`) instead of raw `fmt.Errorf`
- Public functions and structs must have comments
- Table-driven tests preferred
- Test coverage target: >80%

## Testing
- Unit tests alongside source: `*_test.go`
- Integration tests: `test/integration/`
- Security tests: `test/security/`
- Playwright E2E: `web/kubes-aura/tests/`
- Run all: `make test-all`
- Run unit only: `make test-unit`

## Build Verification
Before any commit, ensure:
1. `go build ./...` compiles (known exceptions: proto package, ui dist)
2. `go vet ./...` passes
3. Relevant tests pass
4. `gofmt` applied

## Task Master Workflow
- Set task to `in-progress` before starting
- Create feature branch from develop
- Work through subtasks if expanded
- Update subtask notes with implementation details
- Set task to `done` when complete
- Commit with conventional commit referencing task

## Pre-existing Build Issues (Known)
- `cmd/gpio-test-client/main.go` — missing proto package (needs `make proto`)
- `internal/ui/ui.go` — missing dist files (needs frontend build)
- These are pre-existing and should NOT block other work
