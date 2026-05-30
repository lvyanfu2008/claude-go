# claude-go TUI 前端显示对齐 claude-code 设计文档

## 目标

将 claude-go 的 TUI 前端显示效果与 claude-code (TS) 对齐，分 4 个区域、15 项改动。

---

## 区域 1：整体布局和视觉效果（7 项）

### 1.1 用户提示符字形

- **当前**：`>` (ASCII)
- **目标**：`❯` (U+276F)，与 TS `figures.pointer` 一致
- **文件**：`gou/app/repl_chrome.go` — `UserPromptPointerGlyph()`
- **复杂度**：低

### 1.2 用户提示符缩进

- **当前**：`"  > "`（4 列前缀），续行 `"    "`（4 空格）
- **目标**：`"❯ "`（2 列前缀），续行 `"  "`（2 空格），与 TS 一致
- **文件**：`gou/message/user_message.go` — `styleUserMessageLines()`
- **复杂度**：低

### 1.3 用户消息垂直间距

- **当前**：用户消息被 `"\n" + rows + "\n"` 包裹，上下各一空行
- **目标**：去掉包裹的空行，改为与上下文一致的间距
- **文件**：`gou/message/user_message.go` — `Render()` 返回值
- **复杂度**：低

### 1.4 全局消息窗格槽

- **当前**：`messagePaneGutterCols = 2`，所有消息行向右偏移 2 列
- **目标**：去掉全局槽（设为 0），消息贴左边界渲染
- **文件**：
  - `gou/app/main.go` — `messagePaneGutterCols` 常量修改
  - `gou/app/message_renderer_integration.go` — `applyMessagePaneGutter()` 改为透传
- **复杂度**：中（影响面广，需回归验证各消息类型的渲染）

### 1.5 助手文本块前缀

- **当前**：`"  ⏺ "`（全局槽 2 + 块前缀 3 = 5 列），续行 `"    "`（6 列）
- **目标**：去掉全局槽后，首行 `"⏺ "`（3 列），续行 `"  "`（2 列）
- **文件**：`gou/message/assistant_message.go` — `renderTextBlock()` 前缀常量
- **复杂度**：低（与 1.4 联动）

### 1.6 输入区域下方标尺线

- **当前**：输入区域上下各一条 `─` 标尺（`promptAboveInputRuleLine` 和下方标尺）
- **目标**：仅保留上方标尺，去掉下方标尺
- **文件**：`gou/app/main.go` — `View()` 方法
- **复杂度**：低

### 1.7 紧凑边界字形

- **当前**：`⟳` (U+27F3)
- **目标**：`✻` (U+273B)，与 TS `CompactBoundaryMessage.tsx` 一致
- **文件**：`gou/message/system_message.go` — `renderCompactBoundary()`
- **复杂度**：低

---

## 区域 2：组件化架构（3 项）

### 2.1 Slash Picker 提取为子模型

- **当前**：渲染逻辑在 `submit_aux.go`（`renderSlashPicker()`），键盘处理在 `main.go`（`handleSlashListNavKey()`、`handleKeyMsgPreserving()`），匹配逻辑在 `slash_suggest_ts.go`
- **目标**：新建 `slash_picker.go`，包含 `slashPickerModel` 子模型（`Update()` + `View()`），内聚：
  - 斜杠命令数据加载
  - 过滤/排序逻辑
  - ↑↓/Tab/Enter 键盘处理
  - 可见性控制（F2 切换 + `/` 自动显示 + ESC 关闭）
- **父模型集成**：`main.go` 中 `m.slashPicker.Update(msg)` 替代分散处理
- **复杂度**：中（需要重构分散在 3 个文件中的逻辑）
- **文件**：新建 `gou/app/slash_picker.go`，整合 `submit_aux.go`、`slash_suggest_ts.go`、`slash_result_panel.go` 中相关代码

### 2.2 Permission Modal 提取为子模型

- **当前**：`permissionAskOverlay` 结构体在 `submit_aux.go`，渲染和键盘处理分散
- **目标**：新建 `permission_modal.go`，包含 `permissionModalModel` 子模型：
  - 独立的 `Update()` 处理 Y/N/D/Esc 按键
  - 独立的 `View()` 渲染弹窗
  - 通过 channel 与父模型通信结果
- **复杂度**：中
- **文件**：新建 `gou/app/permission_modal.go`

### 2.3 Message Pane 提取为独立渲染函数

- **当前**：`View()` 方法中 ~50 行 if-else 处理消息窗格渲染（视口路径 vs 非视口路径）
- **目标**：新建 `message_pane.go`，提取为 `renderMessagePane(m *model) string` 函数
- **复杂度**：低
- **文件**：新建 `gou/app/message_pane.go`，`main.go` 中改为调用此函数

---

## 区域 3：消息渲染细节（5 项）

### 3.1 链接渲染

- **当前**：`[text](url)` Markdown 语法原样输出
- **目标**：提取 text 部分，以下划线 + 蓝色渲染（不显示 URL）
- **文件**：`gou/markdown/render.go` — `RenderTokens()` 和 token 解析
- **复杂度**：中（需要 goldmark 扩展或自定义 link 渲染）

### 3.2 表格渲染

- **当前**：Markdown 表格源码原样输出（不可读）
- **目标**：渲染为 Unicode 框线表格（`┌──┬──┐` 风格），对齐 TS
- **文件**：新建 `gou/markdown/table.go` — goldmark 表格扩展的自定义渲染器
- **复杂度**：高（需要处理列宽计算、对齐、ANSI 转义序列转义、内容换行）

### 3.3 水平分隔线

- **当前**：`---` 原样输出
- **目标**：渲染为 faint `─` 标尺行
- **文件**：`gou/markdown/render.go` — 添加 `KindHorizontalRule` token 处理
- **复杂度**：低

### 3.4 图片占位

- **当前**：`![alt](url)` 源码原样输出
- **目标**：渲染为 `[Image: alt]` 占位文本（TS 行为）
- **文件**：`gou/markdown/render.go` — 添加 image token 处理
- **复杂度**：低

### 3.5 代码块语言标签

- **当前**：代码块无语言标签
- **目标**：在代码块首行上方显示语言名称
- **文件**：`gou/markdown/render.go` — 代码块渲染函数
- **复杂度**：低

---

## 区域 4：交互组件

**无新增改动**。四大交互组件（AskUserQuestion、Permission Modal、Transcript、Hooks Config Menu）在之前迭代中已完成与 TS 的对齐。

---

## 涉及文件

| 文件 | 改动项 |
|------|--------|
| `gou/app/main.go` | #4 去槽, #6 去标尺, #8-#10 组件拆分集成 |
| `gou/app/repl_chrome.go` | #1 提示符 |
| `gou/app/submit_aux.go` | #8-#9 拆分 |
| `gou/app/slash_suggest_ts.go` | #8 整合 |
| `gou/app/slash_result_panel.go` | #8 整合 |
| `gou/app/message_renderer_integration.go` | #4 去槽, #10 |
| `gou/message/user_message.go` | #2 缩进, #3 间距 |
| `gou/message/assistant_message.go` | #5 前缀 |
| `gou/message/system_message.go` | #7 字形 |
| `gou/markdown/render.go` | #11 链接, #13 水平线, #14 图片, #15 标签 |
| `gou/markdown/table.go`（新建） | #12 表格 |
| `gou/app/slash_picker.go`（新建） | #8 |
| `gou/app/permission_modal.go`（新建） | #9 |
| `gou/app/message_pane.go`（新建） | #10 |

## 复杂度分布

- 低（10 项）：#1, #2, #3, #5, #6, #7, #10, #13, #14, #15
- 中（4 项）：#4, #8, #9, #11
- 高（1 项）：#12 表格渲染

总计：15 项，涉及 14 个文件（4 个新建）。

## 实施顺序

1. **区域 1（布局）**：#1-#7，先改视觉层
2. **区域 2（组件化）**：#8-#10，再重组结构
3. **区域 3（渲染）**：#11-#15，最后补渲染能力

遵循先易后难、先视觉后结构的原则，每项改动独立可验证。
