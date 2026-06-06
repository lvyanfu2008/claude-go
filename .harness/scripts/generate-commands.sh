#!/bin/bash
# Command Generation Script for Claude Code Spec Workflow (Unix/Linux/macOS)
#
# This script generates individual task commands for each task in a spec's tasks.md file.
# It creates a folder structure under .harness/commands/{spec-name}/ with individual
# command files for each task that call /spec-execute with the appropriate parameters.
#
# Usage: ./generate-commands.sh <spec-name>

set -e

if [ -z "$1" ]; then
    echo "Error: Spec name is required"
    echo "Usage: ./generate-commands.sh <spec-name>"
    exit 1
fi

SPEC_NAME="$1"
PROJECT_ROOT="$(pwd)"
SPEC_DIR="$PROJECT_ROOT/.harness/specs/$SPEC_NAME"
TASKS_FILE="$SPEC_DIR/tasks.md"
COMMANDS_SPEC_DIR="$PROJECT_ROOT/.harness/commands/$SPEC_NAME"

# Check if tasks.md exists
if [ ! -f "$TASKS_FILE" ]; then
    echo "Error: tasks.md not found at $TASKS_FILE"
    exit 1
fi

# Create spec commands directory
mkdir -p "$COMMANDS_SPEC_DIR"

# Parse tasks and generate commands
TASK_COUNT=0
echo "Parsing tasks from $TASKS_FILE..."

generate_task_command() {
    local task_id="$1"
    local task_desc="$2"
    local command_file="$COMMANDS_SPEC_DIR/task-$task_id.md"

    cat > "$command_file" << EOF
# $SPEC_NAME - Task $task_id

Execute task $task_id for the $SPEC_NAME specification.

## Task Description
$task_desc

## Usage
```
/$SPEC_NAME-task-$task_id
```

## Instructions
This command executes a specific task from the $SPEC_NAME specification.

**Automatic Execution**: This command will automatically execute:
```
/spec-execute $task_id $SPEC_NAME
```

**Process**:
1. Load the $SPEC_NAME specification context (requirements.md, design.md, tasks.md)
2. Execute task $task_id: "$task_desc"
3. Follow all implementation guidelines from the main /spec-execute command
4. Mark the task as complete in tasks.md
5. Stop and wait for user review

**Important**: This command follows the same rules as /spec-execute:
- Execute ONLY this specific task
- Mark task as complete by changing [ ] to [x] in tasks.md
- Stop after completion and wait for user approval
- Do not automatically proceed to the next task

## Next Steps
After task completion, you can:
- Review the implementation
- Run tests if applicable
- Execute the next task using /$SPEC_NAME-task-[next-id]
- Check overall progress with /spec-status $SPEC_NAME
EOF
}

# Parse tasks from markdown
while IFS= read -r line; do
    # Match task lines like "- [ ] 1. Task description" or "- [ ] 2.1 Task description"
    if [[ $line =~ ^[[:space:]]*-[[:space:]]*\[[[:space:]]*\][[:space:]]*([0-9]+(.[0-9]+)*)[[:space:]]*\.?[[:space:]]*(.+)$ ]]; then
        task_id="${BASH_REMATCH[1]}"
        task_desc="${BASH_REMATCH[3]}"

        generate_task_command "$task_id" "$task_desc"
        ((TASK_COUNT++))
    fi
done < "$TASKS_FILE"

echo
echo "Generated $TASK_COUNT task commands for spec: $SPEC_NAME"
echo "Commands created in: .harness/commands/$SPEC_NAME/"
echo
echo "Generated commands:"

# Show generated commands
while IFS= read -r line; do
    if [[ $line =~ ^[[:space:]]*-[[:space:]]*\[[[:space:]]*\][[:space:]]*([0-9]+(.[0-9]+)*)[[:space:]]*\.?[[:space:]]*(.+)$ ]]; then
        task_id="${BASH_REMATCH[1]}"
        task_desc="${BASH_REMATCH[3]}"
        echo "  /$SPEC_NAME-task-$task_id - $task_desc"
    fi
done < "$TASKS_FILE"
