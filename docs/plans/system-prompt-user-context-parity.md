# System Prompt / User Context Go-TS 差距分析

TS 侧完整 prompt pipeline: `getSystemPrompt()` → `buildEffectiveSystemPrompt()` → `appendSystemContext()` → `prependUserContext()` → `queryModel()`（含 attribution / prefix / cache_control blocks）

Go 侧完整 prompt pipeline: `BuildGouDemoSystemPrompt()` → `FetchSystemPromptParts()` → `AppendSystemContext()` + `PrependUserContext()` → streaming_loop 以单一字符串发送

## P0: API 调用时 system prompt 以单一字符串发送，无 prompt caching

**TS 行为** (`src/services/api/claude.ts` + `src/utils/api.ts`):

1. `queryModel()` 在 system prompt 前追加 attribution header 和 CLI sysprompt prefix
2. `splitSysPromptPrefix()` 将 system prompt 按 `__SYSTEM_PROMPT_DYNAMIC_BOUNDARY__` 分割为 cache-scoped blocks:
   - Pre-boundary 静态内容 → `cache_control: { type: "ephemeral" }` scope=global
   - Post-boundary 动态内容 → `cache_control: null`
3. system prompt 作为 `content` 数组（每项 `{type: "text", text, cache_control}`）发送

**Go 现状** (`conversation-runtime/query/streaming_loop.go:97`):

```go
req["system"] = strings.Join(in.SystemPrompt, "\n\n")  // 单一字符串, 无 cache_control
```

**差距**:
- 无 `x-anthropic-billing-header` attribution header
- 无 `You are Claude Code, Anthropic's official CLI for Claude.` prefix
- 无 system prompt content blocks 数组, 无 `cache_control` 标记
- `StripSystemPromptDynamicBoundaryForAPI()` 只是去掉 marker, 不是 split + scope

**需要实现**:
1. 在 `streaming_loop.go`（及其他 API loop: openai, gemini, nostream）中, 将 system prompt 从单一字符串改为 `[]SystemPromptBlock` 数组
2. 实现 `splitSysPromptPrefix()` — 按 boundary marker 分割, 给不同部分不同 cache scope
3. 实现 attribution header + CLI sysprompt prefix 注入
4. `messagesapi` 层支持 system prompt content blocks 序列化（含 `cache_control`）

## P1: Coordinator mode 完全缺失

**TS 行为** (`src/coordinator/coordinatorMode.ts`):
- `getCoordinatorSystemPrompt()` — 替代 default system prompt 的 coordinator 系统提示词
- `getCoordinatorUserContext()` — 向 user context 注入 `workerToolsContext` 字段
- 在 `QueryEngine.ts:307` 和 `REPL.tsx:3316` 合并到 user context

**Go 现状**: 无 coordinator 相关代码。

**需要实现**:
1. `commands/coordinator_prompt.go` — coordinator system prompt builder
2. `querycontext/coordinator_context.go` — `workerToolsContext` 注入到 user context
3. `buildEffectiveSystemPrompt` 选择 coordinator prompt 的逻辑

## P2: terminalFocus 未在运行时注入 user context

**TS 行为** (`src/screens/REPL.tsx:741`):
```typescript
if (isProactive) {
  userContext.terminalFocus = terminalFocused ? 'focused' : 'unfocused'
}
```

**Go 现状**: `terminalFocus` 只存在于 proactive system prompt section 的静态文本中 (`commands/prompts.go:700-703`)，未在运行时根据实际终端焦点状态动态注入 user context。

**需要实现**: 在 `BuildUserContext()` 或调用处, 检测终端焦点状态, 向 user context map 注入 `terminalFocus`。

## P3: 无 Advisor tool instructions 动态追加

**TS 行为** (`src/services/api/claude.ts:1425`):
```typescript
...(advisorModel ? [ADVISOR_TOOL_INSTRUCTIONS] : []),
```

**Go 现状**: 不存在。

**需要实现**: 检测 advisor model 是否激活, 在 final system prompt 中追加 `ADVISOR_TOOL_INSTRUCTIONS`。

## P4: 无 Chrome tool search instructions 动态追加

**TS 行为** (`src/services/api/claude.ts:1426`):
```typescript
...(injectChromeHere ? [CHROME_TOOL_SEARCH_INSTRUCTIONS] : []),
```

**Go 现状**: 不存在。

**需要实现**: 检测是否在 Chrome context 中, 追加 `CHROME_TOOL_SEARCH_INSTRUCTIONS`。

---

## 对比总览

| 模块 | TS 文件 | Go 文件 | 状态 |
|------|---------|---------|------|
| Static sections (21) | `constants/prompts.ts` | `commands/prompts.go` + `gou_demo_system.go` | ✅ 对齐 |
| Session guidance | `constants/prompts.ts:353` | `commands/session_guidance.go:74` | ✅ 对齐 |
| Auto memory prompt | `memdir/memdir.ts:419` | `memdir/memdir.go:341` | ✅ 对齐 |
| Environment info | `constants/prompts.ts:652` | `commands/prompts.go:314` | ✅ 对齐 |
| Proactive sections | `constants/prompts.ts:861` | `commands/prompts.go:630-709` | ✅ 对齐 |
| System context (gitStatus, cacheBreaker) | `context.ts:116` | `querycontext/system_context.go:10` | ✅ 对齐 |
| User context (claudeMd, currentDate) | `context.ts:155` | `querycontext/user_context.go:9` | ✅ 对齐 |
| `<system-reminder>` wrapper | `api.ts:452` | `query/user_context.go:42` | ✅ 对齐 |
| MessagesForQuery | `query.ts` | `query/messages_query.go` | ✅ 对齐 |
| Tool result budget | `query.ts:413` | `query/query.go:75` | ✅ 对齐 |
| Auto-compact | `query.ts:488` | `query/query.go:118` | ✅ 对齐 |
| Skill listing delta | `query.ts` | `main.go` (skillListingSent) | ✅ 对齐 |
| Attribution header | `system.ts:73` | `query/system_prompt.go:69` | ✅ P0 已实现 |
| CLI sysprompt prefix | `system.ts:30` | `query/system_prompt.go:63` | ✅ P0 已实现 |
| Content blocks + cache_control | `api.ts:324` + `claude.ts:3285` | `query/system_prompt.go:88` + `streaming_loop.go:97` | ✅ P0 已实现 |
| Coordinator mode user context | `coordinatorMode.ts:80` | `querycontext/coordinator_context.go:27` | ✅ P1 已实现 |
| Advisor instructions | `claude.ts:1425` | `query/system_prompt.go:198` + `streaming_loop.go:104` | ✅ P3 已实现 |
| Chrome instructions | `claude.ts:1426` | `query/system_prompt.go:222` + `streaming_loop.go:107` | ✅ P4 已实现 |
| terminalFocus runtime injection | `REPL.tsx:741` | `querycontext/terminal_focus.go:56` | ✅ P2 已实现 |
