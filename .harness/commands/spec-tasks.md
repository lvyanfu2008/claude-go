# Spec Tasks Command

Generate implementation task list based on approved design.

## Usage
```
/spec-tasks [feature-name]
```

## Instructions
You are working on the tasks phase of the spec workflow.

**WORKFLOW**: This is the FINAL step before command generation.
**SEQUENCE**: Create Tasks → Get Approval → THEN run script
**DO NOT** run any scripts until tasks are approved.

1. **Prerequisites**
   - Ensure design.md exists and is approved
   - Load both requirements.md and design.md for context
   - Understand the complete feature scope

2. **Generate Task List**
   - Break design into atomic, executable coding tasks
   - Use checkbox format with numbered hierarchy
   - Each task should reference specific requirements
   - Focus ONLY on coding tasks (no deployment, user testing, etc.)

3. **Task Guidelines**
   - Tasks should be concrete and actionable
   - Include specific file names and components
   - Build incrementally (each task builds on previous)
   - Reference requirements using _Requirements: X.Y_ format
   - Use test-driven development approach

4. **Task Format**
   ```markdown
   - [ ] 1. Task description
     - Sub-bullet with details
     - Specific files to create/modify
     - _Requirements: 1.1, 2.3_
   ```

5. **Excluded Tasks**
   - User acceptance testing
   - Deployment to production
   - Performance metrics gathering
   - User training or documentation
   - Business process changes

6. **Approval Process**
   - Present the complete task list
   - Ask: "Do the tasks look good?"
   - Make revisions based on feedback
   - Continue until explicit approval

7. **Generate Task Commands** (ONLY after tasks approval)
   - **WAIT**: Do not run script until user explicitly approves tasks
   - **THEN EXECUTE**: `./.harness/scripts/generate-commands-launcher.sh {feature-name}`
   - **PURPOSE**: Creates individual task commands in `.harness/commands/{feature-name}/`
   - **RESULT**: Each task gets its own command: `/{feature-name}-task-{task-id}`
   - **EXAMPLE**: Creates `/{feature-name}-task-1`, `/{feature-name}-task-2.1`, etc.
   - **IMPORTANT**: Do NOT edit the scripts - just run them as-is
   - **PLATFORM**: Automatically detects OS and runs appropriate script (Windows/macOS/Linux)

## Task Structure
```markdown
# Implementation Plan

- [ ] 1. Setup project structure
  - Create directory structure
  - Define core interfaces
  - _Requirements: 1.1_

- [ ] 2. Implement data models
- [ ] 2.1 Create base model classes
  - Define validation methods
  - Write unit tests
  - _Requirements: 2.1, 2.2_
```

## Next Phase
After approval and command generation, you can:
- Use `/spec-execute` to implement tasks
- Use individual task commands: `/{feature-name}-task-1`, `/{feature-name}-task-2`, etc.
- Check progress with `/spec-status {feature-name}`
