# 测试案例文档 - 2026年4月29日提交

## 概述

本文档记录了针对 2026年4月29日前后 4 个提交的测试案例，涉及内存预取（memory prefetch）、内存目录扫描与选择（memdir）、fork subagent 提取重构、以及 extractmemories 重构。

## 提交概览

### 1. `7e134a3` — refactor: extract fork subagent and envTruthy helpers into shared tools/tool package
**变更**: 将 `IsForkSubagentEnabled`、`EnvTruthy` 等辅助函数从 `tools/agent_tool.go` 和 `tools/fork_subagent.go` 提取到共享的 `tools/tool/fork.go` 包中。

### 2. `2fd94ce` — refactor(extractmemories): move memory scanning to memdir; add FindRelevantMemories selector
**变更**: 将内存文件扫描逻辑从 `services/extractmemories` 移至 `memdir` 包，新增 `ScanMemoryFiles`、`FormatMemoryManifest`、`FindRelevantMemories`（调用 Sonnet 选择最多 5 个相关记忆文件）。

### 3. `dcd22f1` — feat(memoryprefetch): add async relevant-memory prefetch consumed after tool results
**新增**: `services/memoryprefetch` 包，实现异步相关记忆预取。在模型循环开始前启动后台 goroutine，轮询结果在工具结果后注入。

### 4. `0ffc19b` — fix(memoryprefetch): simplify getUserMessageText JSON parsing to single-block
**修复**: 简化 `getUserMessageText` 的 JSON 解析逻辑，移除数组内容块解析，改为单对象 `{role, content}` 格式。

---

## 测试案例

### 测试组 1: `tools/tool/fork.go` — EnvTruthy 和 fork subagent 门控函数

#### 测试案例 1.1: EnvTruthy — 各真值字符串
**目标**: 验证 `EnvTruthy` 正确识别 "1"、"true"、"yes"、"on"（大小写不敏感）
**输入**: 分别设置环境变量为上述值
**预期输出**: 均返回 `true`
**边界情况**: 大小写混合（`True`、`YES`、`On`）、前后带空格

#### 测试案例 1.2: EnvTruthy — 假值和其他字符串
**目标**: 验证非真值字符串返回 `false`
**输入**: 空字符串、`"0"`、`"false"`、`"no"`、`"off"`、`"maybe"`、`"enabled"`
**预期输出**: 均返回 `false`

#### 测试案例 1.3: EnvTruthy — 环境变量未设置
**目标**: 验证环境变量不存在时返回 `false`
**输入**: 未设置目标环境变量
**预期输出**: `false`

#### 测试案例 1.4: IsForkSubagentEnabled — 正常交互式会话
**目标**: 验证在普通交互式会话中 fork subagent 启用
**前置条件**: stdout 是 TTY；`CLAUDE_CODE_COORDINATOR_MODE` 未设置或被设为 false；`CLAUDE_CODE_NONINTERACTIVE` / `HEADLESS` / `GOU_DEMO_NON_INTERACTIVE` 均未设置
**预期输出**: `true`

#### 测试案例 1.5: IsForkSubagentEnabled — coordinator 模式禁用
**目标**: 验证 coordinator 模式下禁用 fork subagent
**前置条件**: 设置 `FEATURE_COORDINATOR_MODE=1` 且 `CLAUDE_CODE_COORDINATOR_MODE=1`
**预期输出**: `false`

#### 测试案例 1.6: IsForkSubagentEnabled — 非交互式会话禁用
**目标**: 验证非交互式会话中禁用 fork subagent
**输入**: 分别设置 `CLAUDE_CODE_NONINTERACTIVE=1`、`HEADLESS=1`、`GOU_DEMO_NON_INTERACTIVE=1`
**预期输出**: 均为 `false`
**附加验证**: stdout 非 TTY 时也返回 `false`

#### 测试案例 1.7: coordinatorModeEnvShim — 特性门控
**目标**: 验证 `FEATURE_COORDINATOR_MODE` 未启用时，即使 `CLAUDE_CODE_COORDINATOR_MODE=1` 也返回 `false`
**前置条件**: `FEATURE_COORDINATOR_MODE` 未设置（或为 `0`）
**输入**: `CLAUDE_CODE_COORDINATOR_MODE=1`
**预期输出**: `false`

#### 测试案例 1.8: nonInteractiveSessionEnvShim — 多条件优先级
**目标**: 验证任一非交互条件满足即返回 `true`
**输入**: 仅设置 `HEADLESS=true`，其他条件均为 false
**预期输出**: `true`

---

### 测试组 2: `memdir/find_relevant.go` — 内存文件扫描与选择

#### 测试案例 2.1: ScanMemoryFiles — 正常扫描目录
**目标**: 验证扫描内存目录返回正确的 MemoryHeader 列表
**前置条件**: 临时目录包含若干合法 `.md` 文件（含合法 YAML frontmatter），以及非 `.md` 文件和 `MEMORY.md`
**预期输出**:
- 仅返回 `.md` 文件（排除 `MEMORY.md`）
- 按 mtime 降序排列（最新在前）
- 每个 header 包含正确的 `Filename`、`FilePath`、`Description`、`Type`
- `MEMORY.md` 被过滤掉
- 非 `.md` 文件被过滤掉

#### 测试案例 2.2: ScanMemoryFiles — 无 frontmatter 的文件
**目标**: 验证缺少 frontmatter 的 `.md` 文件仍然被包含
**输入**: 不含 `---` 分隔符的 `.md` 文件
**预期输出**: header 存在，但 `Description` 和 `Type` 为空字符串

#### 测试案例 2.3: ScanMemoryFiles — 空目录
**目标**: 验证空目录（或不存在）返回空列表
**输入**: 空的临时目录
**预期输出**: `len(headers) == 0`

#### 测试案例 2.4: ScanMemoryFiles — 超过 200 文件上限
**目标**: 验证文件数超过 `maxMemoryFiles`(200) 时截断
**输入**: 创建 250 个 `.md` 文件
**预期输出**: 返回恰好 200 个 header（最新的 200 个）

#### 测试案例 2.5: ScanMemoryFiles — 损坏的 YAML frontmatter
**目标**: 验证 YAML 解析失败时不崩溃，优雅降级
**输入**: `.md` 文件包含格式错误的 YAML（如不闭合的引号、错误的缩进）
**预期输出**: header 正常返回，`Description`/`Type` 为空字符串

#### 测试案例 2.6: FormatMemoryManifest — 基本格式化
**目标**: 验证内存 header 列表格式化为清单文本
**输入**: 包含不同 type/description 组合的 header 列表
**预期输出**:
- 有 type 和 description: `- [type] filename (RFC3339时间): description`
- 有 type 无 description: `- [type] filename (RFC3339时间)`
- 无 type 有 description: `- filename (RFC3339时间): description`
- 每行一个 header

#### 测试案例 2.7: FormatMemoryManifest — 空列表
**目标**: 验证空列表返回空字符串
**输入**: `[]MemoryHeader{}`
**预期输出**: `""`

#### 测试案例 2.8: FindRelevantMemories — 已曝光路径过滤
**目标**: 验证 `alreadySurfaced` 参数正确过滤已展示的路径
**前置条件**: 创建 5 个内存文件，其中 2 个路径在 `alreadySurfaced` map 中
**输入**: `alreadySurfaced` 包含 2 个路径
**预期输出**: 仅 3 个未曝光文件被送入 selector

#### 测试案例 2.9: FindRelevantMemories — 全部已曝光
**目标**: 验证所有文件都已曝光时返回空列表
**输入**: `alreadySurfaced` 包含所有文件路径
**预期输出**: `nil`（不调用 API）

#### 测试案例 2.10: FindRelevantMemories — 无 API Key
**目标**: 验证缺少 API Key 时返回 nil（不崩溃）
**输入**: `ANTHROPIC_API_KEY` 和 `ANTHROPIC_AUTH_TOKEN` 均未设置
**预期输出**: `nil` + 错误日志

#### 测试案例 2.11: selectRelevantMemories — API 返回空列表
**目标**: 验证 Sonnet 返回空 `selected_memories` 时的行为
**模拟**: Mock HTTP 返回 `{"content":[{"type":"text","text":"{\"selected_memories\":[]}"}]}`
**预期输出**: 空 `[]string`

#### 测试案例 2.12: selectRelevantMemories — 非法文件名过滤
**目标**: 验证 API 返回的文件名若不在有效列表中则被过滤
**模拟**: Sonnet 返回包含不存在文件名的 `selected_memories`
**预期输出**: 仅包含有效文件名

#### 测试案例 2.13: selectRelevantMemories — API 非 2xx 响应
**目标**: 验证 API 返回错误状态码时正确处理
**模拟**: Mock HTTP 返回 401 / 429 / 500
**预期输出**: 返回 error

#### 测试案例 2.14: selectRelevantMemories — 响应 JSON 解析失败
**目标**: 验证 API 返回非 JSON 或格式错误内容时的行为
**模拟**: 返回无效 JSON
**预期输出**: 返回解析错误

#### 测试案例 2.15: parseFrontmatterYAML — 多行 description
**目标**: 验证多行 description 字段正确解析
**输入**: YAML frontmatter 的 description 包含换行符和多行内容
**预期输出**: description 字符串包含完整内容

---

### 测试组 3: `services/memoryprefetch/` — 异步内存预取

#### 测试案例 3.1: StartRelevantMemoryPrefetch — 正常流程端到端
**目标**: 验证从消息中提取用户输入、选择相关内存、读取文件、构造附件的完整流程
**前置条件**: 内存目录存在且有若干 `.md` 文件；API key 可用
**输入**: 包含至少一条非 meta 用户消息的 messages 列表
**预期输出**:
- 返回非 nil `*Handle`
- 后台 goroutine 启动（channel 有缓冲）
- 不阻塞调用线程

#### 测试案例 3.2: StartRelevantMemoryPrefetch — 自动内存未启用时返回 nil
**目标**: 验证 `IsAutoMemoryEnabled()` 返回 false 时跳过
**前置条件**: 模拟禁用自动内存
**预期输出**: `nil`

#### 测试案例 3.3: StartRelevantMemoryPrefetch — 特性门控关闭时返回 nil
**目标**: 验证 `growthbook.IsTenguMothCopse()` 返回 false 时跳过
**前置条件**: 模拟特性门控关闭
**预期输出**: `nil`

#### 测试案例 3.4: StartRelevantMemoryPrefetch — 无用户消息
**目标**: 验证消息列表中没有非 meta 用户消息时返回 nil
**输入**: messages 仅包含 assistant 消息或 meta 用户消息
**预期输出**: `nil`

#### 测试案例 3.5: StartRelevantMemoryPrefetch — 用户消息文本为空
**目标**: 验证用户消息提取后文本为空字符串时返回 nil
**输入**: 用户消息的 `Message` 和 `Content` 字段均无法解析出文本
**预期输出**: `nil`

#### 测试案例 3.6: StartRelevantMemoryPrefetch — 单词语法（无词边界）
**目标**: 验证单词输入（无空格/制表符/换行）被 `hasWordBoundary` 拒绝
**输入**: 用户消息为 `"hello"`（无空格）
**预期输出**: `nil`
**边界情况**: 纯符号如 `"测试"`（中文无空格）也需要拒绝

#### 测试案例 3.7: StartRelevantMemoryPrefetch — 会话字节数超限
**目标**: 验证先前已曝光的记忆总字节数超过 `MAX_SESSION_BYTES`(60KB) 时跳过
**输入**: 消息列表中已包含大量 `relevant_memories` 附件，累计 content 字节数 > 60KB
**预期输出**: `nil`

#### 测试案例 3.8: Handle.Poll — 结果尚未就绪（channel 未填充）
**目标**: 验证后台任务未完成时 Poll 返回 nil, nil（不阻塞）
**前置条件**: Handle 刚创建，后台 goroutine 仍在运行
**预期输出**: `nil, nil`

#### 测试案例 3.9: Handle.Poll — 结果就绪
**目标**: 验证后台任务完成后 Poll 返回记忆附件消息
**前置条件**: 后台 goroutine 已完成，channel 中有结果
**预期输出**: 返回 `[]types.Message`，包含 1 条 `relevant_memories` 附件消息，最多 5 个记忆

#### 测试案例 3.10: Handle.Poll — 结果已消费（重复调用）
**目标**: 验证第二次 Poll 调用返回 nil
**前置条件**: 第一次 Poll 已消费结果
**预期输出**: `nil, nil`

#### 测试案例 3.11: Handle.Poll — channel 关闭无结果
**目标**: 验证后台任务未找到相关记忆（未写入 channel，channel 关闭时为空）
**前置条件**: FindRelevantMemories 返回空列表
**预期输出**: `nil, nil`（settled 置为 true）

#### 测试案例 3.12: Handle.Poll — 结果截断为 5 个
**目标**: 验证当后台返回超过 5 个记忆时 Poll 截断
**输入**: 后台 goroutine 写入 8 个 SurfacedMemory
**预期输出**: 附件消息仅包含前 5 个记忆

#### 测试案例 3.13: Handle.Close — 中止进行中的请求
**目标**: 验证 Close 正确通知后台 goroutine 停止
**前置条件**: Handle 已创建，后台 goroutine 正在运行（阻塞在 `select` 等待 channel 写入）
**操作**: 调用 `h.Close()`
**预期**: 后台 goroutine 的 `<-done` 分支被选中，不写入 channel

#### 测试案例 3.14: Handle.Close — nil handle 安全
**目标**: 验证对 nil Handle 调用 Close 不会 panic
**输入**: `(*Handle)(nil).Close()`
**预期输出**: 无 panic

#### 测试案例 3.15: Handle.Close — 重复关闭安全
**目标**: 验证对同一 Handle 多次调用 Close 不会 panic（done channel 已关闭）
**操作**: `h.Close(); h.Close()`
**预期输出**: 无 panic

---

### 测试组 4: `services/memoryprefetch/helpers.go` — 辅助函数

#### 测试案例 4.1: getLastUserMessage — 多条消息中查找最后一条
**目标**: 验证从混合消息列表中找到最后一条非 meta 用户消息
**输入**: messages = `[user(meta), assistant, user(non-meta), assistant, user(meta)]`
**预期输出**: 第二个用户消息（索引 2 处的 non-meta user）

#### 测试案例 4.2: getLastUserMessage — 无用户消息
**目标**: 验证所有消息都是 assistant 时返回 false
**输入**: 全部为 assistant 消息
**预期输出**: `ok == false`

#### 测试案例 4.3: getLastUserMessage — IsMeta 为 nil vs false
**目标**: 验证 `IsMeta` 为 nil（未设置）的消息被当作非 meta 处理
**输入**: 用户消息 `IsMeta = nil`
**预期输出**: 该消息被选中

#### 测试案例 4.4: getUserMessageText — 字符串内容
**目标**: 验证 `Message` 字段为 JSON 字符串时的解析
**输入**: `Message` = `json.RawMessage("\"hello world\"")`
**预期输出**: `"hello world"`（去除前后空白）

#### 测试案例 4.5: getUserMessageText — 单对象块（`{role, content}`）
**目标**: 验证修复后的解析逻辑（commit `0ffc19b`）
**输入**: `Message` = `json.RawMessage("{\"role\":\"user\",\"content\":\"你好\"}")`
**预期输出**: `"你好"`

#### 测试案例 4.6: getUserMessageText — 两个字段都无法解析
**目标**: 验证 `Message` 和 `Content` 都不是合法 JSON 字符串/对象时返回空
**输入**: `Message` = 空的 `json.RawMessage`，`Content` = 空的 `json.RawMessage`
**预期输出**: `""`

#### 测试案例 4.7: getUserMessageText — 优先使用 Message 字段
**目标**: 验证 `Message` 非空时优先使用，不回退到 `Content`
**输入**: `Message` = `"\"from message\""`, `Content` = `"\"from content\""`
**预期输出**: `"from message"`

#### 测试案例 4.8: getUserMessageText — Message 为空时回退到 Content
**目标**: 验证 `Message` 为空时使用 `Content`
**输入**: `Message` = `nil`/`""`, `Content` = `"\"from content\""`
**预期输出**: `"from content"`

#### 测试案例 4.9: readMemoryFile — 正常读取（不超出限制）
**目标**: 验证文件内容小于 line/byte 限制时完整返回
**输入**: 50 行、2KB 的文件
**预期输出**: 完整内容，`truncated == false`

#### 测试案例 4.10: readMemoryFile — 行数超限
**目标**: 验证行数超过 `MAX_MEMORY_LINES`(200) 时截断并添加截断提示
**输入**: 300 行的文件
**预期输出**: 前 200 行 + 截断提示消息

#### 测试案例 4.11: readMemoryFile — 字节数超限
**目标**: 验证字节数超过 `MAX_MEMORY_BYTES`(4096) 时截断并添加截断提示
**输入**: 10KB 的单行文件（行数不超但字节超）
**预期输出**: 前 4096 字节 + 截断提示

#### 测试案例 4.12: readMemoryFile — 同时超限
**目标**: 验证行数和字节数同时超限时的行为（先按字节截断，再按行截断）
**输入**: 500 行、50KB 的文件
**预期输出**: 先取前 4096 字节，再取前 200 行，包含截断提示

#### 测试案例 4.13: readMemoryFile — 文件不存在
**目标**: 验证读取不存在的文件时返回错误
**输入**: 不存在的文件路径
**预期输出**: `error != nil`

#### 测试案例 4.14: readMemoriesForSurfacing — 部分文件读取失败
**目标**: 验证当多个文件中有个别读取失败时跳过失败的继续处理
**输入**: 3 个 RelevantMemory，其中 1 个路径不存在
**预期输出**: 返回 2 个 SurfacedMemory（失败的被跳过，不中断）

#### 测试案例 4.15: collectSurfacedMemories — 提取已曝光路径和字节数
**目标**: 验证从消息列表中正确累计已曝光的记忆
**输入**: 3 条消息，其中 2 条为 `relevant_memories` 附件，包含不同路径的记忆
**预期输出**: `paths` map 包含所有不重复路径；`totalBytes` 为所有记忆 content 长度之和

#### 测试案例 4.16: collectSurfacedMemories — 无附件消息
**目标**: 验证消息列表无 relevant_memories 附件时返回空
**输入**: 全部为普通 user/assistant 消息
**预期输出**: `paths` 为空 map，`totalBytes == 0`

#### 测试案例 4.17: newRelevantMemoriesAttachment — 基本构造
**目标**: 验证附件消息正确序列化
**输入**: 2 个 SurfacedMemory
**预期输出**:
- `Type == MessageTypeAttachment`
- `IsMeta == true`
- `Attachment` 反序列化后 `type == "relevant_memories"`
- `memories` 数组包含 2 个元素

#### 测试案例 4.18: memoryAge — 今天/昨天/N 天前
**目标**: 验证时间格式化
**输入**: mtime = 当前时间（today）、24h 前（yesterday）、5 天前
**预期输出**: `"today"`、`"yesterday"`、`"5 days ago"`

#### 测试案例 4.19: memoryAgeDays — 未来的 mtime
**目标**: 验证 mtime 在未来时返回 0（非负数）
**输入**: mtime = 明天的时间戳
**预期输出**: `0`

#### 测试案例 4.20: memoryFreshnessText — 新鲜记忆
**目标**: 验证 1 天内的记忆不显示陈旧警告
**输入**: mtime = 今天或昨天
**预期输出**: `""`（空字符串）

#### 测试案例 4.21: memoryFreshnessText — 陈旧记忆
**目标**: 验证超过 1 天的记忆显示陈旧警告
**输入**: mtime = 7 天前
**预期输出**: 包含 "This memory is 7 days old" 和 "point-in-time observations" 的完整警告文本

#### 测试案例 4.22: memoryHeader — 新鲜 vs 陈旧
**目标**: 验证 header 根据新鲜度生成不同格式
**输入**:
- 新鲜记忆: mtime = 今天
- 陈旧记忆: mtime = 30 天前
**预期输出**:
- 新鲜: `"Memory (saved today): path:"`
- 陈旧: 先陈旧警告文本，然后 `"\n\nMemory: path:"`

---

### 测试组 5: `memdir/memdir.go` — 重构后的 extractmemories 依赖

#### 测试案例 5.1: BuildMemoryFrontmatterExample — 格式验证
**目标**: 验证 frontmatter 示例包含必要字段
**预期输出**: 返回的行数组中包含 `---` 分隔符、`name:`、`description:`、`type:`

#### 测试案例 5.2: BuildTypesSectionIndividual — 包含所有类型
**目标**: 验证类型分类包含 user / feedback / project / reference 四种类型
**预期输出**: 返回的行数组中包含 `<name>user</name>`、`<name>feedback</name>` 等标记

#### 测试案例 5.3: BuildWhatNotToSaveSection — 排除规则
**目标**: 验证 "不应保存到记忆" 的条目包含代码模式、git 历史等排除项
**预期输出**: 包含 "Code patterns" 或 "Git history" 等关键词

---
