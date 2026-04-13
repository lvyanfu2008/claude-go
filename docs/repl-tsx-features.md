# REPL.tsx 功能清单（对照用）

参考源码：[`claude-code/src/screens/REPL.tsx`](../../claude-code/src/screens/REPL.tsx)（约 6k 行，Ink + React）。

**标注说明**

- **【主要】**：构成「主会话 REPL 壳」的核心能力；缺了会明显不像产品主界面。
- **【次要】**：增强、分支形态、条件编译（`feature()` / `USER_TYPE` / env），或周边通知类。

Go 侧拆分与范围边界见 [`gou/replparity`](../gou/replparity/doc.go) 包注释、[`plans/gou-demo-repl-bridge-scope.md`](plans/gou-demo-repl-bridge-scope.md)。

---

## 1. 屏幕与布局【主要】

| 功能 | 说明 |
|------|------|
| 双模式 `Screen = 'prompt' \| 'transcript'`（非双物理屏） | 主会话 vs 转写视图；进入转写时冻结 `messagesLength` + `streamingToolUsesLength`（TS `frozenTranscriptState`） |
| `AlternateScreen`、`FullscreenLayout` | 交替屏、高度约束、与转写虚拟滚动分支共用 |
| `AnimatedTerminalTitle`、`useTerminalTitle` | 动态终端标题；可 `CLAUDE_CODE_DISABLE_TERMINAL_TITLE` |
| 虚拟滚动 `VirtualMessageList` + `scrollRef` + `useVirtualScroll` | TS：`VirtualMessageList` 内用 `useVirtualScroll` 算可见区间；`scrollRef` 为 Ink `ScrollBoxHandle`（`scrollTo` / `scrollToBottom` / `getScrollTop` 等）。Go：[`goc/gou/virtualscroll`](../gou/virtualscroll/virtual_scroll.go) 对应 `useVirtualScroll`；[`cmd/gou-demo/main.go`](../cmd/gou-demo/main.go) 的 `scrollTop` / `sticky` / `pendingDelta` / `heightCache` + `View` / `refineVisibleHeights` 对应「列表 + 命令式滚动」的合并效果（无独立 ref 对象）。**鼠标**：`tea.WithMouseCellMotion`，消息区滚轮与左键拖拽改 `scrollTop`（[`mouse_message_list.go`](../cmd/gou-demo/mouse_message_list.go)）；`GOU_DEMO_DISABLE_MOUSE_SCROLL=1` 关闭。`CLAUDE_CODE_DISABLE_VIRTUAL_SCROLL=1` 时放宽单次挂载上限（调试向，仍非整表 Ink ScrollBox）；见 `gouDemoVirtualScrollDisabled` |

---

## 2. 消息与对话【主要】

| 功能 | 说明 |
|------|------|
| `messages` / `setMessages`、`deferredMessages` | 对话列表；流式时用 `useDeferredValue` 减轻 Messages 协调卡顿。Go 无对等 API；gou-demo 对 **`assistant_delta` 跳过全量 `rebuildHeightCache`**（只改 `StreamingText`，正文在虚拟列表外渲染），**prompt 下 `gouStreamingToolUsesMsg` 跳过全量 rebuild**（流式工具行不在 scroll keys），转写模式仍 rebuild。见 [`stream_ui_height.go`](../cmd/gou-demo/stream_ui_height.go) |
| `Messages` 组件 | 各类 Message、工具块、折叠组等渲染；**列表数据源管线**（progress / null 附件 / `shouldShowUserMessage` / `reorderMessagesInUI` / 非虚拟转写尾窗）见 [`goc/gou/messagesview`](../gou/messagesview/doc.go)，gou-demo 经 [`messagesForScroll()`](../cmd/gou-demo/transcript_screen.go) 接入 |
| `streamingText`、`onStreamingText`、`handleMessageFromStream` | 流式增量、与 `query()` 对接 |
| `streamingToolUses`、`streamingThinking` | 流式工具调用、thinking；Go gou-demo 侧对应 [`goc/gou/conversation.Store`](../gou/conversation/store.go) 的 `StreamingToolUses`（HTTP 流式由 [`query.QueryDeps.OnStreamingToolUses`](../conversation-runtime/query/params.go) 经 Bubble Tea 消息同步，ccb NDJSON 仍多为空） |
| `toolJSX`、本地 JSX slash（`/model`、`/mcp` 等） | 全屏底部模态、与 `FullscreenLayout` 联动 |
| `unseenDivider`、`jumpToNew` | 新消息分割线与跳转 |

---

## 3. 输入与提交【主要】

| 功能 | 说明 |
|------|------|
| `PromptInput` | 多模式、`vimMode`、`PromptInputQueuedCommands` |
| `handlePromptSubmit` / `onSubmit` | `processUserInput`、`query` 管线入口 |
| `history`、`parseReferences`、`expandPastedTextRefs` | 历史、引用、粘贴展开 |
| `userInputOnProcessing`、`placeholderText` | 提交后至首包占位 |

---

## 4. 模型查询与工具【主要】

| 功能 | 说明 |
|------|------|
| `query()` 主循环 | 调模型、流式、工具执行编排 |
| `abortController`、`CancelRequestHandler` | 取消进行中的请求（如 Ctrl+C） |
| `useMergedTools`、`assembleToolPool`、`mergeAndFilterTools` | 工具合并与过滤 |
| `useMergedCommands`、`commands`、slash | 与输入、权限联动 |
| `MCPConnectionManager`、`dynamicMcpConfig` | MCP 连接与动态配置 |

---

## 5. 权限与安全【主要】

| 功能 | 说明 |
|------|------|
| `PermissionRequest`、`toolUseConfirmQueue` | 工具执行前确认 |
| Sandbox：`SandboxManager`、`SandboxPermissionRequest`、`SandboxViolationExpandedView` | 沙箱与违规展示 |
| `ToolPermissionContext`、`applyPermissionUpdate`、bypass 检查 | 权限模式与自动降级 |
| Swarm/Worker：`WorkerPendingPermission`、`registerLeaderToolUseConfirmQueue` 等 | 多代理权限桥 |

---

## 6. 转写模式（Transcript）【主要】

| 功能 | 说明 |
|------|------|
| 进入/退出、`frozenTranscriptState` | 冻结 `messages` 与 `streamingToolUses` 长度 |
| `GlobalKeybindingHandlers`（如 `ctrl+o`） | 与全局快捷键表一致 |
| `ScrollKeybindingHandler` | j/k、PgUp/PgDn、空格、`g`/`G` 等 |
| `TranscriptSearchBar`、`useSearchHighlight`、`jumpRef`、`reorderMessagesInUI` | `/` 搜索、`n`/`N`、Esc 与高亮；列表与搜索 haystack 顺序同 TS 分组；Go：`messagesForScroll` + `highlightSearchPlain`（非 dump）。高亮为 **lipgloss 子串包裹**（ANSI），与 Ink/React DOM 高亮实现不同，语义对齐「可见文本命中加亮」 |
| `[` dump、`v` 外部编辑器 | `renderMessagesToPlainText`、临时文件、`openFileInExternalEditor` |
| `ctrl+e`、`showAllInTranscript` | 展开 collapsed / grouped |
| `TranscriptModeFooter` | 转写底部状态与快捷键提示 |

---

## 7. ReplBridge / 远程 / SSH【次要】

| 功能 | 说明 |
|------|------|
| `useReplBridge` | 远程 REPL / 控制面消息 |
| `useRemoteSession`、`useDirectConnect`、`useSSHSession` | 远程会话形态 |

> Go `gou-demo` 明确不实现 ReplBridge 客户端，见 [`plans/gou-demo-repl-bridge-scope.md`](plans/gou-demo-repl-bridge-scope.md)。

---

## 8. Swarm / 队友 / 本地 Agent【次要】

| 功能 | 说明 |
|------|------|
| `TeammateViewHeader`、`viewingAgentTaskId` | 查看队友/子代理会话 |
| `injectUserMessageToTeammate`、`InProcessTeammateTask` | 向队友注入用户消息 |
| `TaskListV2`、`useTasksV2WithCollapseEffect` | 任务列表与折叠 |
| `useBackgroundTaskNavigation` | Shift+Down 等与后台任务对话框 |
| `useTeammateViewAutoExit` | 队友结束自动退出查看模式 |

---

## 9. 会话生命周期与存储【主要】

| 功能 | 说明 |
|------|------|
| `conversationId`、标题、`generateSessionTitle` | 会话标识与命名 |
| 恢复：`deserializeMessages`、`restoreSessionStateFromLog`、`copyPlanForResume` 等 | 断点续聊、计划恢复 |
| Compact：`partialCompactConversation`、`runPostCompactCleanup` | 上下文压缩 |
| File history：`fileHistoryMakeSnapshot`、`fileHistoryRewind` | 文件级撤销 |
| Content replacement：`provisionContentReplacementState` 等 | 工具结果持久化与回放 |
| `ExitFlow`、`gracefulShutdownSync` | 退出清理 |

---

## 10. IDE / 安装 / 选择【次要】

| 功能 | 说明 |
|------|------|
| `useIdeSelection`、`IdeOnboardingDialog` | IDE 选择与扩展安装引导 |
| `closeOpenDiffs`、`getConnectedIdeClient` | 与 IDE 协同 |

---

## 11. 通知、调查、增长【次要】

| 功能 | 说明 |
|------|------|
| 各类 `use*Notification` | 限流、弃用、npm、IDE 状态、模型迁移、订阅等 |
| `useFeedbackSurvey`、`useMemorySurvey`、`usePostCompactSurvey` | 反馈与问卷 |
| `useFrustrationDetection`（ant）、`useAntOrgWarningNotification`（ant） | 内部构建 |
| `useInstallMessages`、`useChromeExtensionNotification` 等 | 安装与市场提示 |

---

## 12. 成本与计费【次要】

| 功能 | 说明 |
|------|------|
| `useCostSummary`、`CostThresholdDialog`、`showCostDialog` | 成本阈值与确认 |
| `IdleReturnDialog` | 空闲返回 |
| Token 预算：`getCurrentTurnTokenBudget`、`snapshotOutputTokensForTurn` 等 | 与 bootstrap state 联动 |

---

## 13. 其它交互与装饰【次要】

| 功能 | 说明 |
|------|------|
| `MessageSelector` | 选择用户消息重发等 |
| `CompanionSprite`（`BUDDY`） | 伙伴形象 |
| `SpinnerWithVerb`、`BriefIdleStatus` | 加载态与空闲一行状态 |
| `VoiceKeybindingHandler`（`VOICE_MODE`） | 语音 |
| `WebBrowserPanel`（`WEB_BROWSER_TOOL`） | 浏览器工具面板 |
| `TungstenLiveMonitor` | Tungsten 工具监控 |
| `useProactive` / `useScheduledTasks`（`PROACTIVE`/`KAIROS`） | 主动任务 |
| `autoUpdaterResult` | 自动更新结果 |
| `AwsAuthStatusBox` | AWS 认证状态条 |

---

## 14. 键盘与命令总线【主要】

| 功能 | 说明 |
|------|------|
| `KeybindingSetup`、`GlobalKeybindingHandlers`、`CommandKeybindingHandlers` | 全局与应用命令键位 |
| `MessageActionsKeybindings`（`MESSAGE_ACTIONS`） | 消息区光标与动作 |

---

## 15. 一句话归纳

- **【主要】**：双屏 + 消息列表与流式 + 输入提交 + `query` 与工具/MCP + 权限/沙箱 + 转写（搜索/导出/展开）+ 会话存储与压缩/恢复 + 全局键位。
- **【次要】**：ReplBridge/远程、Swarm/队友、大量通知与问卷、成本弹窗、IDE、语音/浏览器/主动式等。

更新本清单时：对照 `REPL.tsx` 顶部 import 与 `export function REPL` 内 JSX 分支（`screen === 'transcript'` 与主 `mainReturn`）。
