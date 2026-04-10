> **已废弃（架构假设）**  
> 下文采用 **「TS 终端 UI + 壳不变，Go 仅承接后半段」** 的过渡划分，与当前 **Go 持有 TUI 与编排主路径** 不一致。  
> **权威架构**见 **[../docs/plans/architecture-go-orchestration.md](../docs/plans/architecture-go-orchestration.md)**。  
> 保留原文仅供历史讨论参考。

---

## 不要参考下文作当前架构决策

这是一种典型的 「TS 终端 UI + 壳 / 输入不变，Go 承接后半段」 划分。可以按下面方式处理（与仓库里 architecture-go-ts-strategy.md、ccb-engine 思路一致）。

1. 边界画在哪里
UI 与「用户输入 → 会话里多什么」尽量仍在 TS
REPL / PromptInput / handlePromptSubmit / processUserInput（含 processSlashCommand）可以不改或极少改：继续产出 newMessages、shouldQuery、allowedTools、model/effort 等。
Go 从「决定要调模型这一轮」开始接手
等价于现在 onQueryImpl 里在拼好 messages + systemPrompt + userContext + systemContext + toolUseContext 里与请求相关的部分 之后，不再走 TS query()，改为调用 Go 服务（进程内包或独立 ccb-engine 类 sidecar）。
也就是说：数据处理在方案里 = turn 编排 + 调模型 + 流式解析 +（可选）工具调度；不是把 processUserInput 硬搬进 Go，除非你愿意维护双份 slash 语义。

2. 推荐数据流（概念上）
TS（原 UI 路径）
用户提交 → executeUserInput → processUserInput → 合并 messages、得到 shouldQuery。
若 shouldQuery == false
仍全部在 TS 结束（本地斜杠、纯 UI 等），不必碰 Go。
若 shouldQuery == true
TS 组装 「本轮请求快照」（见下），通过 稳定 IPC（stdio / UDS / HTTP、gRPC 等，与现有 Go 协议对齐）发给 Go。
Go
执行流式 API、维护本轮状态机、产出 assistant / tool_use / tool_result 等事件。
工具与权限
策略 A（与当前策略文档一致）：工具仍在 TS（文件、LSP、IDE…），Go 收到 tool_use 后 回调 TS 执行，再把 tool_result 塞回 Go 继续生成。
策略 B：仅把「无 UI、纯 HTTP」的工具迁到 Go，其余仍回调 TS。
TS UI
只负责 订阅事件（流式 token、工具进度、权限弹窗）：数据从 Go（或经 TS 薄代理）推回 现有 setMessages / store，Ink 组件尽量不动。
3. TS → Go 要带什么「快照」
至少对齐现有 QueryParams 思路，便于少改 UI：

消息列表（含本轮新消息）
systemPrompt + userContext + systemContext（或 Go 侧只收 hash + 关键字段，由约定决定）
工具列表 / schema（或工具 ID 与 TS 注册表一致）
AbortController 的取消语义（在协议里对应 cancel / context cancel）
协议版本号（避免双端语义漂移）
权限 canUseTool：若在 TS 弹窗，则 Go 发 「需要批准」 事件 → TS 展示现有权限 UI → 把结果回传 Go。

4. 和「ProcessSlashCommand 保持 TS」如何共存
整段输入解析仍在 TS：/foo 仍在 processSlashCommand 里跑完，Go 看不到原始 / 串也没关系。
**只有「需要模型的一轮」**才把 已解析好的 Message[] + 上下文 交给 Go。
若未来有个别路径想从 Go 调 TS slash：那是 可选适配器，不是 UI 必改项。
5. 实施顺序（务实）
先固定 IPC 消息格式 + 版本（可对照 go/ccb-engine/spec、现有 goEngine TS 客户端）。
TS 只改 onQueryImpl（或等价的一条分支）：在调用 query() 处增加 「走 Go」 门控 + fallback。
Go 先实现 单轮流式 + 无工具或仅 echo 工具，再接入 TS 工具回调。
UI 最后动：优先保持 setMessages / 事件形状与现在一致，减少 Ink 改动。
一句话：UI 与原输入/斜杠逻辑留在 TS；在「准备调用 query() 的那一刻」把会话与上下文打成版本化快照交给 Go；Go 负责模型与（可选）编排，工具与权限通过回调 TS 复用现有实现。 这样既满足「UI 层原代码基本不变」，又满足「后面数据处理用 Go」。

