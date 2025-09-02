# Pi Controller Development Workflow with Task Master

This document outlines the development workflow for completing the Pi Controller Phase 2 implementation using Task Master AI for project management and task tracking.

## Prerequisites

1. **Install Task Master AI globally:**
   ```bash
   npm install -g task-master-ai
   ```

2. **Initialize Task Master in project root:**
   ```bash
   cd /path/to/pi-controller
   task-master init
   ```

3. **Parse the PRD to generate tasks:**
   ```bash
   task-master parse-prd .taskmaster/docs/prd.txt --research
   ```

4. **Analyze task complexity and expand where needed:**
   ```bash
   task-master analyze-complexity --research
   task-master expand --all --research
   ```

## Development Workflow

### Daily Startup

1. **Check next available tasks:**
   ```bash
   task-master next
   ```

2. **View detailed task information:**
   ```bash
   task-master show <task-id>
   ```

3. **Start working on a task:**
   ```bash
   task-master set-status --id=<task-id> --status=in-progress
   ```

### During Implementation

1. **Log progress and implementation notes:**
   ```bash
   task-master update-subtask --id=<subtask-id> --prompt="Implementation progress: added GPIO pin control logic, tested with mock hardware"
   ```

2. **Update task with discoveries/blockers:**
   ```bash
   task-master update-task --id=<task-id> --prompt="Discovered need for hardware abstraction layer, added to architecture"
   ```

3. **Add new tasks discovered during development:**
   ```bash
   task-master add-task --prompt="Implement hardware safety checks for GPIO operations" --research
   ```

### Code Quality Workflow

Before marking any task as complete, ensure code quality standards:

1. **Run linting and fix issues:**
   ```bash
   make install-lint # First time only
   make lint
   make vet
   make fmt
   ```

2. **Run relevant tests:**
   ```bash
   make test-unit # For service layer changes
   make test-gpio # For GPIO-related changes
   make test-security # For security-related changes
   make test-api # for API changes
   ```

3. **Check build success:**
   ```bash
   make build
   ```

4. **Only after all checks pass, mark task complete:**
   ```bash
   task-master set-status --id=<task-id> --status=done
   ```

### File-Level Development Pattern

For each file that needs implementation:

1. **Create comprehensive task covering all changes needed:**
   ```bash
   task-master add-task --prompt="Complete pi-agent GPIO implementation in cmd/pi-agent/main.go - add hardware interface, gRPC communication, system monitoring, configuration handling, graceful shutdown" --research
   ```

2. **Expand into specific subtasks:**
   ```bash
   task-master expand --id=<task-id> --research --force
   ```

3. **Work through subtasks systematically:**
   - Implement functionality
   - Add tests
   - Update documentation
   - Run linting and fix issues
   - Verify build success

4. **Log detailed progress:**
   ```bash
   task-master update-subtask --id=<subtask-id> --prompt="Implemented gRPC client connection with retry logic and health checking"
   ```

## Priority Implementation Order

Based on the PRD analysis, tackle tasks in this order:

### Phase 2.1: Foundation (High Priority)

```bash
# Focus on these task types first:
task-master list --status=pending | grep -i "code quality\|foundation\|infrastructure"
```

### Phase 2.2: Hardware Control (High Priority)

```bash
# Focus on GPIO and pi-agent implementation:
task-master list --status=pending | grep -i "gpio\|pi-agent\|hardware"
```

### Phase 2.3: Discovery Service (High Priority)

```bash
# Focus on node discovery and networking:
task-master list --status=pending | grep -i "discovery\|mdns\|network"
```

## Testing Strategy

### Unit Testing Pattern

For each implemented component:

1. **Write tests first (TDD approach):**
   ```bash
   task-master add-task --prompt="Write comprehensive unit tests for GPIO service methods"
   ```

2. **Run tests during development:**
   ```bash
   make test-unit
   ```

3. **Achieve coverage targets:**
   ```bash
   make test-coverage-threshold # Must be >80%
   ```

### Integration Testing

For cross-component functionality:

1. **Create integration test tasks:**
   ```bash
   task-master add-task --prompt="Integration test for pi-agent to controller GPIO communication via gRPC"
   ```

2. **Run integration tests:**
   ```bash
   make test-integration
   ```

### Security Testing

For security-sensitive components:

1. **Run security tests:**
   ```bash
   make test-security-verbose
   ```

2. **Address any vulnerabilities immediately:**
   ```bash
   task-master add-task --prompt="Fix security vulnerability identified in authentication middleware"
   ```

## Code Quality Gates

No task should be marked as `done` until:

1. **All linting passes:**
   ```bash
   make lint # Must exit with code 0
   ```

2. **No vet errors:**
   ```bash
   make vet # Must exit with code 0
   ```

3. **Tests pass:**
   ```bash
   make test-unit # All tests green
   ```

4. **Build succeeds:**
   ```bash
   make build # Must build without errors
   ```

5. **Coverage maintained:**
   ```bash
   make test-coverage-threshold # Must be >80%
   ```

## Task Documentation

### Research-Backed Development

For complex tasks, use research mode to get AI assistance:

```bash
task-master expand --id=<task-id> --research # Get research-backed subtasks
task-master update-task --id=<task-id> --prompt="Need guidance on Kubernetes controller implementation" --research
```

### Implementation Logging

Document implementation decisions and learnings:

```bash
task-master update-subtask --id=<subtask-id> --prompt="Used periph.io/x/periph for GPIO access instead of sysfs for better performance and safety"
```

### Blocking Issues

When encountering blockers:

```bash
task-master update-task --id=<task-id> --prompt="Blocked: Need real Raspberry Pi hardware for GPIO testing, using mock implementation for now"
task-master add-task --prompt="Set up Raspberry Pi test hardware environment"
```

## Progress Tracking

### Weekly Review

Check overall progress:

```bash
task-master list # See all tasks
task-master complexity-report # Review task complexity analysis
```

### Sprint Planning

Plan upcoming work:

```bash
task-master list --status=pending # See remaining work
task-master next # Get next priority task
```

### Milestone Tracking

Track phase completion:

```bash
# Count completed tasks by phase
task-master list --status=done | grep "Phase 2.1" | wc -l
task-master list --status=pending | grep "Phase 2.1" | wc -l
```

## Common Commands Reference

### Task Management

```bash
task-master list # List all tasks
task-master show <id> # Show task details
task-master next # Get next task
task-master set-status --id=<id> --status=<status> # Update status
task-master add-task --prompt="<description>" # Add new task
task-master update-task --id=<id> --prompt="<update>" # Update task
```

### Development Workflow

```bash
make lint && make vet && make test-unit && make build # Quality check
make test-all # Full test suite
make clean && make build-all # Clean rebuild
```

### Task Status Values

- `pending` - Ready to work on
- `in-progress` - Currently being implemented
- `review` - Implementation complete, needs verification
- `done` - Completed and verified
- `blocked` - Cannot proceed due to dependency
- `deferred` - Postponed for later implementation

## Best Practices

1. **One file, one comprehensive task** - Don't split single file changes across multiple tasks
2. **Always run quality checks** - Lint, vet, test before marking complete
3. **Document implementation decisions** - Use update-subtask to log key decisions
4. **Use research mode for complex tasks** - Get AI assistance for architecture decisions
5. **Test incrementally** - Run tests after each significant change
6. **Keep tasks focused** - Each task should have a clear, measurable outcome
7. **Log blockers immediately** - Don't let blockers delay progress tracking

This workflow ensures systematic completion of the Pi Controller Phase 2 implementation while maintaining high code quality and comprehensive documentation of the development process.
