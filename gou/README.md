# gou — Go TUI 基础库（对话流 + 虚拟滚动）

**权威产品架构**（Go：TUI、消息处理、skill 加载、LLM 编排）见 **[`docs/plans/architecture-go-orchestration.md`](../../docs/plans/architecture-go-orchestration.md)**。`goc` **默认构建与运行不依赖** Bun/Node 执行 TS；仓库内嵌入数据与注释中的 TS 路径不算运行时依赖（与根目录 [`CLAUDE.md`](../CLAUDE.md)「No TypeScript runtime dependency」一致）。

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
| `goc/gou/messagerow` | `SegmentsFromMessage` — content 块 + `grouped_tool_use` / `collapsed_read_search` + `server_tool_use` / `advisor_tool_result`；**`collapsed_read_search`** 摘要文案由 [`SearchReadSummaryText`](messagerow/search_read_summary.go) 对齐 TS [`getSearchReadSummaryText`](../../claude-code/src/utils/collapseReadSearch.ts)，行末附带 **[`CtrlOToExpandHint`](messagerow/search_read_summary.go)**（与 Ink `CtrlOToExpand` 字面一致）。gou-demo 下 **`ctrl+o`** 绑定 **transcript 全屏**（冻结进入时刻的消息列表；`Esc`/`q`/`ctrl+c` 关闭，`ctrl+e` 切换 show-all 提示），见 [`docs/plans/gou-demo-transcript-ts-parity.md`](../docs/plans/gou-demo-transcript-ts-parity.md)。 |
| `goc/gou/transcript` | 从 JSON 加载消息：UI 形 `[]Message`（含 `type`）或 API 形 `[{role,content}]` |
| `goc/gou/ccbstream` | 将 ccb-engine 风格 NDJSON `StreamEvent` 应用到 `conversation.Store`（`Apply` / `Feed` / `ReplayFile`） |
| `goc/conversation-runtime/query` | gou-demo **真实模型**：HTTP 流式 parity（`StreamingParity` + env 门控 + 密钥）；`-fake-stream` 为纯模拟 |
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
| 真实 LLM（gou-demo） | `goc/conversation-runtime/query` 流式 parity | **已做**：`ANTHROPIC_API_KEY`（或 `ANTHROPIC_AUTH_TOKEN`）+ 宿主打开流式 parity（`ApplyQueryHostEnvGates` / `QueryParams.StreamingParity`）；可选 `GOU_QUERY_STREAMING_PARITY=1`；`-fake-stream` / `GOU_DEMO_USE_FAKE_STREAM` 为纯模拟；未配置时 **降级** 假 `streamTick` 并 system 提示。 |
| `process-user-input` CLI（stdin/stdout JSON） | `goc/cmd/process-user-input` | **可选**（测试 / 自动化；gou TUI 走进程内 `ProcessUserInput`，不依赖 spawn 该二进制） |
| `result.execution` / `result.executionSequence`（bash/slash prepare 桩） | `bashprepare` / `slashprepare` 仍可能填充 `Execution` | **TUI 未执行**（仅 system 提示）；**已移除** Go 独用的 `attachments_plan` / `hooks_plan` / `query` 分支 |
| Ink 级 UI（权限、工具块交互、chroma 等） | `REPL.tsx` 等 | **部分**：transcript（ctrl+o、/ 搜索、ctrl+e 展开、冻结滚动、**`[`** 滚动条 dump、**`v`** 外编辑器）；其余见 [gou-demo-transcript-ts-parity.md](../docs/plans/gou-demo-transcript-ts-parity.md) |

## ProcessUserInput（gou-demo + `goc/gou/pui`）：已做 vs 未做

与 TS 类型 **`ProcessUserInputBaseResult`**（[`src/conversation-runtime/processUserInput/processUserInput.ts`](../../src/conversation-runtime/processUserInput/processUserInput.ts)）对齐说明：

| 字段（TS 名） | 行为 |
|---------------|------|
| `messages` | **已做**：`ApplyProcessUserInputBaseResult` 追加到 `conversation.Store`（不放在 Handoff 结构里）。 |
| `shouldQuery` / `allowedTools` / `model` / `effort` / `resultText` / `nextInput` / `submitNextInput` | **已做**：写入 `ProcessUserInputBaseResultHandoff`（`json` 标签与 TS 一致）；`ApplyProcessUserInputBaseResultOutcome` 带 `effectiveShouldQuery`、`hadExecutionRequest`（Go 侧物化语义，无 TS 同名 struct）。 |
| `execution` / `executionSequence`（如 bash/slash prepare 路径） | **TUI 未执行**：仅 system 说明并清空 Handoff；**不再**从 prompt 路径产生 `attachments_plan` / `hooks_plan` / `query`。 |
| `statePatchBatch` / `hooksReducerInput`（Go 扩展） | **未接**：TUI 不消费。 |

其它约定：

- **`Enter`**：`BuildDemoParams` 组装 `ProcessUserInputParams`（**`uuid`**、`PromptInputModePrompt`、`SkipAttachments`、最小 `ProcessUserInputContextData`）。普通 prompt 与 TS 一样走 **`ProcessTextPrompt`**；**`ExecuteUserPromptSubmitHooks`** 未注入时不在 base 内发 `query` execution。可选：`LogEvent` → **`tengu_input_prompt`**；**`CLAUDE_DEBUG_PROCESS_USER_INPUT`** → stderr **`[processUserInput:…]`**；**`FindCommand`** 用 **`commands` / `runtimeContext.options.commands`**；注入 **`ProcessBashCommand` / `ProcessSlashCommand`** 可替代 bash/slash 的 prepare **`Execution`** 路径；注入 **`ExecuteUserPromptSubmitHooksIter`**（`iter.Seq2`）可按件应用 hook 结果（对齐 TS `for await`）；若与 **`ExecuteUserPromptSubmitHooks`** 同时设置，**优先** Iter。
- **斜杠**：gou-demo 在 `Enter` 提交时调用 `ProcessUserInput` 并注入 **`ProcessSlashCommand`**（`NewSlashResolveProcessSlashCommand`）。**`F2`** 打开本地 slash 名称列表（来自 `GetCommands`）；仍不执行 bash/slash 的 **`Execution`** 桩（见上表）。历史 **`SlashSkippedMessage`** 路径已不在 demo 主流程使用。
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

操作：↑↓ / PgUp / PgDn 滚动消息区，`End` 粘底，`Enter` 发送（**`Ctrl+J` / `Alt+Enter`** 换行，**`Shift+↑↓`** 行间移动光标），**`F2`** slash 列表（打开后可直接输入缩小候选；首行 `/foo` 会作为初始 filter）。**`Ctrl+o`** 进入/退出 **transcript**（冻结历史；transcript 内 **`Esc` / `q` / `Ctrl+c`** 在无 **搜索条** 时仅退出 transcript；搜索条打开时 **`Esc`** 清空搜索留在 transcript；**`/`** 打开搜索、**`Enter`** 收起搜索条并保留查询以便 **`n`/`N`** 跳匹配；列宽变化会清空搜索）。无搜索条且非 **dump** 时 **`j`/`k`** 逐行、**`g`** 顶、**`G`/`Shift+g`** 底、**`Ctrl+u`/`Ctrl+d`** 半屏、**`Ctrl+b`/`Ctrl+f`**、**`b`** 与 **空格** 整屏下翻（**`modalPagerAction`**）、**`Ctrl+n`/`Ctrl+p`** 逐行下/上（与 TS 一致；搜索条打开时这些键不滚动，对齐 **`isModal={!searchOpen}`**）。**`[`**（无搜索条）：TS 风格 **dump**。**`v`**：临时文件 + **`$VISUAL`/`$EDITOR`**。**`Ctrl+e`** 在 dump 下禁用。在 **prompt** 界面 **`q` / `Esc`** 仍退出 demo。未设置 `GOU_QUERY_ASK_STRATEGY=allow` 时，工具权限 **ask** 在 TUI 内以 **Y/N** 模态处理。

**与 Ink REPL 壳层对齐（轻量）**：列宽不足 **80** 列时使用更短的顶栏与底栏提示（对标 TS `columns < 80` / `isNarrow`）。**终端标签标题** 通过 **OSC 0** 设为 `gou-demo`（可带会话 id 截断）；流式进行中标题加 **`…` 前缀**；设置 **`CLAUDE_CODE_DISABLE_TERMINAL_TITLE=1`** 时不写标题序列（与 TS 一致）。底栏可显示 **`CLAUDE_CODE_PERMISSION_MODE`**（如 `plan`、`bypassPermissions`）的短标签与符号（对标 `permissionModeSymbol` / `shortTitle`）。Kitty 下若存在 **`KITTY_WINDOW_ID`**，标题序列使用 **ST** 结尾而非 BEL。

**主题**：合并后的环境变量 **`CLAUDE_CODE_THEME=light`** 使用高对比调色（见 `goc/gou/theme`）。**`GOU_DEMO_STATUS_LINE=1`** 在输入区上方显示一行状态（theme / 消息数 / 列宽等）。工具块中的 `http(s)://` 会做 **OSC 8 超链接**（`goc/gou/textutil.LinkifyOSC8`）。

**调试日志**：`GOU_DEMO_LOG_FILE=/path/to.log` 追加写入；或 `GOU_DEMO_LOG=1` 在 **stderr 为 TTY** 时默认写入 `~/.claude/debug/gou-demo-trace.txt`（全屏 TUI 与 stderr 混用会错位，故不用 stderr）；`GOU_DEMO_LOG_STDERR=1` 强制 stderr。行前缀 `[gou-demo]`。

### 与真实 transcript / NDJSON 流接轨

```bash
# 从 JSON 载入历史（跳过内置 seed）
cd goc && go run ./cmd/gou-demo -transcript=/path/to/messages.json

# 回放已录制的 ccb-engine NDJSON 事件文件，再进入 TUI
cd goc && go run ./cmd/gou-demo -replay-cc=/path/to/stream.ndjson

# 管道读 stdin 上的 NDJSON（Unix 下会尝试打开 /dev/tty 供键盘）
cd goc && cat /path/to/stream.ndjson | go run ./cmd/gou-demo -stream-stdin

# 真实 LLM：API key + 流式 parity（见上表）；密钥可放在 ~/.claude/settings.json 或项目 .claude/settings.go.json
cd goc && go run ./cmd/gou-demo
# 仅 UI 模拟流、不调模型：
cd goc && go run ./cmd/gou-demo -fake-stream
```

**`execute_tool`**：`ccbstream.Apply` 不会执行客户端工具，但会追加一条 **system** 占位说明（工具名、`tool_use_id`），便于管道/回放时看见流里曾有过待执行工具；完整对话形状仍需在同一条 NDJSON 流中提供对应的 **`tool_result`**（或由宿主注入）。

### Go `init.ts` 对齐（进行中）

gou-demo 入口**默认**调用 [`goc/claudeinit`](../claudeinit) 的 **`Init`**（内含 `settingsfile.EnsureProjectClaudeEnvOnce`）。矩阵与缺口见 **[`docs/plans/go-init-port.md`](../../docs/plans/go-init-port.md)**。

对照：`goc/cmd/claude-init-dump`（**Go-only**，适合 CI）。若本地另有 `claude-code` 检出，可 **人工** 运行其 `dump-init-state` + `scripts/compare-init-dumps.sh` 做差异对照（**非** `goc` 必需步骤）。

### 全量 TS system prompt / commands / tools（已移除 Bun 启动快照）

原先的 **`GOU_DEMO_TS_CONTEXT_BRIDGE`** + **`bun run go-context-bridge`**（`claude-code/scripts/go-context-bridge.ts`）已删除。gou-demo 默认使用 **Go 侧** `querycontext` / `commands` 拼装与 **`GOU_DEMO_USE_EMBEDDED_TOOLS_API`**（或 MCP JSON）对齐工具元数据；单元测试仍可向 `pui.DemoConfig.TSContextBridge` / `querycontext.FetchOpts.TSSnapshot` 注入内存中的 [`tscontext.Snapshot`](../tscontext/snapshot.go)。

**Bundled / 磁盘 skill 的 `/` 命令**：gou-demo 在 [`pui/slash_resolve_demo.go`](pui/slash_resolve_demo.go) 中 **进程内** 解析——**磁盘** skill 走 [`goc/slashresolve`](../slashresolve) **`ResolveDiskSkill`**（`SkillRoot` + `SKILL.md`），**bundled** prompt 走 **`ResolveBundledSkill`**（嵌入 markdown）；与 [`goc/commands`](../commands) 加载的 `Command` 列表一致，**不**起 Bun/Node。

**未知斜杠（TS 对齐可选）**：默认将未识别的 `/name` 整行当作普通用户 prompt（`ShouldQuery` 走 `ProcessTextPrompt`）。设置 **`GOU_DEMO_SLASH_STRICT_UNKNOWN=1`** 时，对「看起来像合法命令名」且根路径 **`/name`** 不是已存在文件系统节点的情况，返回 **`Unknown skill: name`** 且不调模型（对齐 `processSlashCommand.tsx` 的 `looksLikeCommand` 分支）。
