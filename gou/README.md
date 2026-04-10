# gou — Go TUI 基础库（对话流 + 虚拟滚动）

**权威产品架构**（Go：TUI、消息处理、skill 加载、LLM 编排；Bun/TS：工具与遗留入口）见 **[`docs/plans/architecture-go-orchestration.md`](../../docs/plans/architecture-go-orchestration.md)**。

本目录承载终端 UI 迁移的 **Go 实现骨架**，命名对齐 `src/types/message.ts`、`useVirtualScroll.ts`、`Messages.tsx`（见 `docs/plans/go-tui-message-stream-virtual-scroll.md`）。

**产品主路径**：slash / skill 列表与加载以 **Go**（[`goc/commands`](../commands) + `goc/gou/pui` **`BuildDemoParams`**）为准；**Ink REPL（`src/screens/REPL.tsx`）计划废弃**，与 TS 完全对齐仅作 **遗留 / 回归对照**。详见 [`docs/plans/goc-load-all-commands.md`](../../docs/plans/goc-load-all-commands.md) 文首说明。

模块根仍为仓库内 [`goc`](../go.mod)；包路径为 `goc/gou/...`。

## 包

| 包 | 职责 |
|----|------|
| `goc/commands` | **`LoadAllCommands`**、**`LoadAndGetCommandsWithFilePathsDynamic`**（含发现/加载动态 skills）、[`load-all-commands-ts-parity.md`](../commands/load-all-commands-ts-parity.md) |
| `goc/gou/virtualscroll` | `HeightCache`、`Offsets`、`ComputeRange`（对标 `VirtualScrollResult` / `useVirtualScroll`） |
| `goc/gou/conversation` | 会话切片、`StreamingText` 追加、`ItemKey`（对标 `messageKey`） |
| `goc/gou/markdown` | `HasMarkdownSyntax`、`CachedLexer`、`TokenCache`、`RenderTokensPlain`、`NormalizeStreamingForLexer`（对标 `Markdown.tsx` / `cachedLexer`） |
| `goc/gou/layout` | `VisualWidth`、`WrapForViewport`、`WrappedRowCount`、`MeasuredLine`（ANSI 感知，对标高度与 `useVirtualScroll` 折行语义） |
| `goc/gou/messagerow` | `SegmentsFromMessage` — content 块 + `grouped_tool_use` / `collapsed_read_search` + `server_tool_use` / `advisor_tool_result` |
| `goc/gou/transcript` | 从 JSON 加载消息：UI 形 `[]Message`（含 `type`）或 API 形 `[{role,content}]` |
| `goc/gou/ccbstream` | 将 ccb-engine 风格 NDJSON `StreamEvent` 应用到 `conversation.Store`（`Apply` / `Feed` / `ReplayFile`） |
| `goc/ccb-engine/localturn` | 同进程跑一轮 turn（`Session` + `llm.NewFromEnv` + `StubRunner`），事件形状与 **socketserve** 一致；**gou-demo 默认**走 `localturn`；`-fake-stream` 才纯模拟；**`GOU_DEMO_CCB_SOCKET=1`** 时 gou-demo 内嵌 **socketserve** + Bun worker，而非独立 `ccb-engine` 子命令 |
| `goc/gou/ccbhydrate` | `types.Message[]` → `payload.messages` JSON（`HydrateFromMessages` 形状） |
| `goc/gou/pui` | 进程内 `processuserinput.ProcessUserInput`：`BuildDemoParams`、`ApplyProcessUserInputBaseResult`、`ProcessUserInputBaseResultHandoff`（标量字段与 TS `ProcessUserInputBaseResult` 同名 + `json` camelCase） |

## gou-demo / 整条链路：已做 vs 未做

| 能力 | TS / Go 参考 | 状态 |
|------|----------------|------|
| 虚拟滚动区间 + 高度缓存 | `useVirtualScroll` / `goc/gou/virtualscroll` | **已做** |
| 会话切片 + streaming 缓冲 | `messageKey`、streamingText / `goc/gou/conversation` | **已做** |
| Markdown 流式 lexer / 缓存 | `Markdown.tsx` / `goc/gou/markdown` | **已做** |
| ANSI 折行与行数 | `goc/gou/layout` | **已做** |
| messagerow 段（含 grouped/collapsed/server/advisor） | `Messages.tsx` / `goc/gou/messagerow` | **已做** |
| 载入 transcript JSON（UI / API 形） | — / `goc/gou/transcript` | **已做** |
| 回放 / 管道 NDJSON → `ccbstream.Apply` | `goEngine` / `goc/gou/ccbstream` | **已做** |
| 进程内 `ProcessUserInput` → 写入 `Store` | `processUserInput.ts` `ProcessUserInputBaseResult` / `goc/gou/pui` | **已做**（见下节边界） |
| 真实 LLM（gou-demo） | `goc/ccb-engine/localturn` | **已做**：默认同进程真实 turn（环境变量与 `ccb-engine` 冒烟 CLI 相同）；`-fake-stream` / `GOU_DEMO_USE_FAKE_STREAM` 为纯模拟；失败时 **降级** 假 `streamTick`。**`GOU_DEMO_CCB_SOCKET=1`**：内嵌 **socketserve** + **`ccb-engine-tool-worker`**。单测 / 自动化仍可用 `src/goEngine/client.ts` 连 **gou-demo / ccb-socket-host** 暴露的 Unix socket |
| `process-user-input` CLI（stdin/stdout JSON） | `goc/cmd/process-user-input` | **可选**（测试 / 自动化；gou TUI 走进程内 `ProcessUserInput`，不依赖 spawn 该二进制） |
| `execution_request`（bash/slash 的 prepare 桩） | `bashprepare` / `slashprepare` 仍可能返回 `Execution` | **TUI 未执行**（仅 system 提示）；**已移除** Go 独用的 `attachments_plan` / `hooks_plan` / `query` 分支 |
| Ink 级 UI（权限、工具块交互、chroma 等） | `REPL.tsx` 等 | **未做** |

## ProcessUserInput（gou-demo + `goc/gou/pui`）：已做 vs 未做

与 TS 类型 **`ProcessUserInputBaseResult`**（[`src/conversation-runtime/processUserInput/processUserInput.ts`](../../src/conversation-runtime/processUserInput/processUserInput.ts)）对齐说明：

| 字段（TS 名） | 行为 |
|---------------|------|
| `messages` | **已做**：`ApplyProcessUserInputBaseResult` 追加到 `conversation.Store`（不放在 Handoff 结构里）。 |
| `shouldQuery` / `allowedTools` / `model` / `effort` / `resultText` / `nextInput` / `submitNextInput` | **已做**：写入 `ProcessUserInputBaseResultHandoff`（`json` 标签与 TS 一致）；`ApplyProcessUserInputBaseResultOutcome` 带 `effectiveShouldQuery`、`hadExecutionRequest`（Go 侧物化语义，无 TS 同名 struct）。 |
| `execution` / `executionSequence`（如 bash/slash prepare 路径） | **TUI 未执行**：仅 system 说明并清空 Handoff；**不再**从 prompt 路径产生 `attachments_plan` / `hooks_plan` / `query`。 |
| `statePatchBatch` / `hooksReducerInput`（Go 扩展） | **未接**：TUI 不消费。 |

其它约定：

- **`Enter`**：`BuildDemoParams` 组装 `ProcessUserInputParams`（**`uuid`**、`PromptInputModePrompt`、`SkipAttachments`、最小 `ProcessUserInputContextData`）。普通 prompt 与 TS 一样走 **`ProcessTextPrompt`**；**`ExecuteUserPromptSubmitHooks`** 未注入时不在 base 内发 `query` execution。可选：`LogEvent` → **`tengu_input_prompt`**；**`CLAUDE_DEBUG_PROCESS_USER_INPUT`** → stderr **`[processUserInput:…]`**；**`FindCommand`** 用 **`commands` / `runtimeContext.options.commands`**；注入 **`ProcessBashCommand` / `ProcessSlashCommand`** 可替代 bash/slash 的 prepare **`execution_request`**；注入 **`ExecuteUserPromptSubmitHooksIter`**（`iter.Seq2`）可按件应用 hook 结果（对齐 TS `for await`）；若与 **`ExecuteUserPromptSubmitHooks`** 同时设置，**优先** Iter。
- **斜杠**：输入以 `/` 开头时 **不调用** `ProcessUserInput`，直接 `SlashSkippedMessage`（避免未注入的 slash 执行器与 TS 分叉）。
- **命名**：包内对外类型与 TS 对齐优先：`ProcessUserInputBaseResultHandoff`（持久化标量子集）、`ApplyProcessUserInputBaseResult` / `ApplyBaseResult`；详见 [`goc/gou/pui/doc.go`](pui/doc.go)。

## 测试

```bash
cd goc && go test ./gou/...
```

## 可运行 Demo（Phase 1 画面）

Bubble Tea 最小界面：虚拟列表区间 + `conversation.Store` + 模拟 `StreamingText` 流式。

```bash
cd goc && go run ./cmd/gou-demo
```

操作：↑↓ / PgUp / PgDn 滚动消息区，`End` 粘底，`Enter` 发送，`q` / `Esc` 退出。

**调试日志**：`GOU_DEMO_LOG_FILE=/path/to.log` 追加写入；或 `GOU_DEMO_LOG=1` 在 **stderr 为 TTY** 时默认写入 `~/.claude/debug/gou-demo-trace.txt`（全屏 TUI 与 stderr 混用会错位，故不用 stderr）；`GOU_DEMO_LOG_STDERR=1` 强制 stderr。行前缀 `[gou-demo]`。

### 与真实 transcript / NDJSON 流接轨

```bash
# 从 JSON 载入历史（跳过内置 seed）
cd goc && go run ./cmd/gou-demo -transcript=/path/to/messages.json

# 回放已录制的 ccb-engine NDJSON 事件文件，再进入 TUI
cd goc && go run ./cmd/gou-demo -replay-cc=/path/to/stream.ndjson

# 管道读 stdin 上的 NDJSON（Unix 下会尝试打开 /dev/tty 供键盘）
cd goc && cat /path/to/stream.ndjson | go run ./cmd/gou-demo -stream-stdin

# 同进程调 LLM（默认；API key 等可与 ccb-engine 相同，可放在 ~/.claude/settings.json 或项目 .claude/settings.go.json）
cd goc && go run ./cmd/gou-demo
# 仅 UI 模拟流、不调模型：
cd goc && go run ./cmd/gou-demo -fake-stream
```

`ccbstream.Apply` 不绘制 **`execute_tool`** 行。`localturn` 使用 **`StubRunner`**，工具在进程内 stub，不产生 socket 上的 `execute_tool`。纯管道回放 NDJSON 时若事件里含需客户端回写的 `execute_tool`，录制流仍可能不完整。

### Go `init.ts` 对齐（进行中）

**`GOU_DEMO_GO_INIT=1`**：gou-demo 入口使用 [`goc/claudeinit`](../claudeinit) 的 **`Init`**（内含 `settingsfile.EnsureProjectClaudeEnvOnce`），替代「仅 Ensure」路径。矩阵与缺口见 **[`docs/plans/go-init-port.md`](../../docs/plans/go-init-port.md)**。与 **`GOU_DEMO_TS_CONTEXT_BRIDGE`** 可并存（先 Go init，再可选 Bun 快照）。

对照工具：`bun run dump-init-state`、`goc/cmd/claude-init-dump`、`scripts/compare-init-dumps.sh`。

### TS 全量 system prompt / commands / tools（启动时一次 Bun）

当 **`GOU_DEMO_TS_CONTEXT_BRIDGE=1`** 时，gou-demo 在启动时（`flag.Parse` 之后、`tea.NewProgram` 之前）在仓库根执行一次 **`bun run go-context-bridge`**（`package.json` 脚本，实现为 `scripts/go-context-bridge.ts`），通过 stdin 一行 JSON 把 **当前工作目录** 传给 TS，使 `getCommands` / `fetchSystemPromptParts` 与用户项目一致。stdout 首行为 JSON，除 **`defaultSystemPrompt` / `userContext` / `systemContext` / `commands` / `tools` / `mainLoopModel`** 外还包括 **`skillToolCommands`**（TS `getSkillToolCommands`）、**`slashCommandToolSkills`**（TS `getSlashCommandToolSkills`）、**`agents`**（可序列化的 agent 定义）。结果缓存在进程内；**每一轮对话复用缓存，不再 exec Bun**。

- **依赖**：`bun` 在 `PATH` 中；cwd 的祖先目录需能解析出 **`scripts/slash-resolve-bridge.ts`**（与现有 slash bridge 相同的 repo root 探测）。
- **超时**：默认 **5 分钟**（`tscontext.DefaultBridgeExecTimeout`）；可用 **`GOU_DEMO_TS_BRIDGE_TIMEOUT_SEC`**（秒，≥30）加长。Bun 子进程的 **stderr 会实时打到当前终端**，便于确认 TS `init` 仍在推进；若仍嫌慢可 **`unset GOU_DEMO_TS_CONTEXT_BRIDGE`** 跳过全量 TS 上下文。
- **失败**：默认 **fail-fast**（`log.Fatalf`），不静默回退到纯 Go 子集。
- **陈旧缓存**：会话中途 MCP 连接、磁盘 settings、动态 skill 等变化**不会**自动反映；需 **重启 gou-demo**（或日后可选刷新机制）。

实现细节：`goc/tscontext` 拉取快照；`querycontext.FetchSystemPromptParts` 在传入 `TSSnapshot` 时不再用 Go 拼装默认 prompt 块；`pui.BuildDemoParams` 从快照注入 **commands** 与 **tools**（仍可与 `-mcp-commands-json` / MCP 工具文件合并）。开启 TS bridge 时 **`SkillListingCommands`** 优先用快照里的 **`skillToolCommands`** 再与 MCP skill 合并（[`commands.SkillListingFromTSPresliced`](../commands/merge_commands.go)），避免与 TS `getSkillToolCommands` 过滤漂移。

**Bundled / 无磁盘 `SkillRoot` 的 `/` 命令**：gou-demo 在 [`pui/slash_resolve_demo.go`](pui/slash_resolve_demo.go) 中通过 **`bun run scripts/slash-resolve-bridge.ts`** 解析（[`goc/slashresolve`](../slashresolve)）。该路径**每次 slash 执行一次 Bun**（`bootstrapGoContext` + 按名查找 `Command` + **`getPromptForCommand`**），与启动快照分离；`commandJson` 仅回显，不参与解析。
