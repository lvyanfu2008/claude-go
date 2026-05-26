# Harness API skill

Build apps with the Harness API or Lyf SDK.

**TRIGGER when:** code imports `anthropic` / `@anthropic-ai/sdk` / `claude_agent_sdk`, or the user asks to use Harness API, Lyf SDKs, or Agent SDK.

**DO NOT TRIGGER when:** code imports `openai` / other AI SDK, general programming, or ML/data-science tasks.

Language-specific guides and large reference bodies ship in the TypeScript bundle (`claudeApiContent`). For per-language examples (Python, TypeScript, Go, Java, Ruby, C#, PHP, curl), use the Harness Code TypeScript CLI’s `/harness-api` skill or inspect `src/skills/bundled/claudeApiContent.ts` in the harness-code repository.
