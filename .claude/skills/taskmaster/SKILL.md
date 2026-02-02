---
description: Comprehensive TaskMaster workflow assistant for managing tasks, parsing PRDs, and tracking development progress
tags: [project, taskmaster]
---

# TaskMaster Workflow Assistant

Interactive workflow for working with TaskMaster AI to manage project tasks efficiently.

## Usage

```
/taskmaster [action] [args]
```

**Actions:**

- `next` - Get and display the next available task to work on
- `show <id>` - Show detailed information about a specific task
- `list [status]` - List all tasks, optionally filtered by status (pending/in-progress/done/blocked)
- `update <id>` - Update a task with new information or implementation notes
- `done <id>` - Mark a task as completed
- `start <id>` - Mark a task as in-progress
- `parse <file>` - Parse a PRD file and generate tasks
- `expand <id>` - Expand a task into subtasks
- `status` - Show project status and task summary
- `analyze` - Analyze task complexity
- `help` - Show this help message

## Workflow Examples

### Starting Your Work Session

```
/taskmaster next
```

This will:

1. Find the next available task based on dependencies and priority
2. Display task details including description, acceptance criteria, and test strategy
3. Suggest the first implementation step

### Working on a Task

```
/taskmaster start 82.1
```

Then as you work, update your progress:

```
/taskmaster update 82.1
# Prompts you to add implementation notes about what you've done
```

### Completing a Task

```
/taskmaster done 82.1
```

This will:

1. Verify the task is complete
2. Mark it as done
3. Show the next available task

### Parsing New Requirements

```
/taskmaster parse .taskmaster/docs/prd-new-feature.txt
```

This will:

1. Parse the PRD file with research mode enabled
2. Generate tasks and append them to the existing task list
3. Show a summary of created tasks

### Viewing All Tasks

```
/taskmaster list
# or filter by status
/taskmaster list pending
/taskmaster list in-progress
```

### Analyzing Project Complexity

```
/taskmaster analyze
```

This will:

1. Analyze all tasks for complexity
2. Identify tasks that need expansion
3. Show complexity report with recommendations

## Task Status Values

- `pending` - Ready to work on
- `in-progress` - Currently being worked on
- `done` - Completed and verified
- `blocked` - Waiting on external factors
- `deferred` - Postponed
- `cancelled` - No longer needed

## Tips

1. **Update Frequently**: Use `/taskmaster update <id>` to log progress, challenges, and decisions as you work
2. **One Task at a Time**: Only have one task in-progress per agent/session
3. **Check Dependencies**: Some tasks are blocked until their dependencies are complete
4. **Use Research Mode**: When parsing PRDs or expanding tasks, research mode provides better results
5. **Review Before Done**: Ensure all acceptance criteria are met before marking tasks as done

## MCP Integration

This skill uses the Task Master MCP server. Available MCP tools:

- `mcp__task-master-ai__get_tasks` - List all tasks
- `mcp__task-master-ai__next_task` - Get next available task
- `mcp__task-master-ai__get_task` - Get specific task details
- `mcp__task-master-ai__set_task_status` - Update task status
- `mcp__task-master-ai__update_task` - Update task information
- `mcp__task-master-ai__update_subtask` - Update subtask notes
- `mcp__task-master-ai__parse_prd` - Parse PRD files
- `mcp__task-master-ai__expand_task` - Expand tasks into subtasks
- `mcp__task-master-ai__analyze_project_complexity` - Analyze complexity

## Arguments Reference

When you run `/taskmaster [action]`, I will:

### For `next`

1. Call `mcp__task-master-ai__next_task` to find the next available task
2. Call `mcp__task-master-ai__get_task` to get full details
3. Display the task with formatting
4. Suggest first implementation steps

### For `show <id>`

1. Call `mcp__task-master-ai__get_task` with the provided ID
2. Display full task details including:
   - Title and description
   - Priority and status
   - Dependencies
   - Acceptance criteria
   - Test strategy
   - Subtasks (if any)

### For `list [status]`

1. Call `mcp__task-master-ai__get_tasks` with optional status filter
2. Display tasks in a formatted table
3. Show task counts by status

### For `update <id>`

1. Prompt you for update details
2. Call `mcp__task-master-ai__update_task` or `mcp__task-master-ai__update_subtask`
3. Confirm the update

### For `done <id>`

1. Call `mcp__task-master-ai__set_task_status` with status "done"
2. Show next available task

### For `start <id>`

1. Call `mcp__task-master-ai__set_task_status` with status "in-progress"
2. Display task details

### For `parse <file>`

1. Call `mcp__task-master-ai__parse_prd` with:
   - `input`: provided file path
   - `append`: true (to add to existing tasks)
   - `research`: true (for better task generation)
2. Show summary of generated tasks

### For `expand <id>`

1. Call `mcp__task-master-ai__expand_task` with:
   - `id`: provided task ID
   - `research`: true
   - `force`: false
2. Show generated subtasks

### For `status`

1. Call `mcp__task-master-ai__get_tasks`
2. Calculate and display:
   - Total tasks
   - Tasks by status
   - Completion percentage
   - Current task (if any in-progress)

### For `analyze`

1. Call `mcp__task-master-ai__analyze_project_complexity` with `research: true`
2. Display complexity analysis results
3. Show recommendations for task expansion

### For `help`

Display this help message with available commands and usage examples.

## Project Context

This skill is aware of the pi-controller project structure and will:

- Use the correct project root: `/Users/spenceryork/Projects/pi-controller`
- Work with the `master` tag by default
- Enable research mode for better AI-powered task generation
- Follow the project's GitFlow branching strategy
- Respect task dependencies and priorities

## Related Documentation

- Task Master CLAUDE.md: `.taskmaster/CLAUDE.md`
- PRD Documentation: `.taskmaster/docs/README.md`
- Project Instructions: `CLAUDE.md`
