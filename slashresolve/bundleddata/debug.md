# Debug Skill

Help the user debug an issue they're encountering in this current Claude Code session.

## Session Debug Log

The debug log for the current session is at: `/Users/lvyanfu/.claude/debug/e480dfdf-3714-463c-a8b9-e0f335f5fb72.txt`

No debug log exists yet — logging was just enabled.

For additional context, grep for [ERROR] and [WARN] lines across the full file.

## Issue Description

The user did not describe a specific issue. Read the debug log and summarize any errors, warnings, or notable issues.

## Settings

Remember that settings are in:
* user - /Users/lvyanfu/.claude/settings.json
* project - /Users/lvyanfu/Work/claude/claude-code/.claude/settings.json
* local - /Users/lvyanfu/Work/claude/claude-code/.claude/settings.local.json

## Instructions

1. Review the user's issue description
2. The last 20 lines show the debug file format. Look for [ERROR] and [WARN] entries, stack traces, and failure patterns across the file
3. Consider launching the claude-code-guide subagent to understand the relevant Claude Code features
4. Explain what you found in plain language
5. Suggest concrete fixes or next steps
