# 测试案例文档 - 2026年4月17日提交

## 概述
本文档记录了针对2026年4月17日三个主要提交的测试案例。这些提交涉及 OpenAI 流式适配器的增强和 gou-demo 的流式工具渲染优化。

## 提交概览

### 1. `0a1060a` - feat(query): add ReplayOpenAIStreamChatResponse for SSE replay (deepseek-reasoner)
**功能**: 添加 `ReplayOpenAIStreamChatResponse` 函数，支持重放录制的 OpenAI SSE 响应（包括 DeepSeek Reasoner 的推理内容）

### 2. `2510dd2` - fix(query): non-stream OpenAI parity replays reasoning_content for DeepSeek reasoner
**修复**: 非流式 OpenAI 兼容性重放现在支持 DeepSeek Reasoner 的推理内容

### 3. `68b1798` - feat(gou-demo): make streaming tool progressive reveal fully non-blocking
**功能**: 使流式工具渐进显示完全非阻塞，移除剩余的 `time.Sleep` 调用

---

## 测试案例

### 测试组 1: OpenAI 流式适配器

#### 测试案例 1.1: ReplayOpenAIStreamChatResponse - 基础文本流
**目标**: 验证基本的 OpenAI SSE 响应重放功能
**输入**: 包含纯文本内容的 SSE 数据
**预期输出**:
- 正确解析 SSE 数据
- 生成正确的 Anthropic 消息流事件
- 包含 `message_start` 和 `content_block_delta` 事件
- 文本内容正确传递

**测试代码参考**:
```go
func TestReplayOpenAIStreamChatResponse_textOnly(t *testing.T) {
    sse := "data: {\"choices\":[{\"index\":0,\"delta\":{}}],\"model\":\"deepseek-chat\"}\n\n" +
           "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hello\"}}]}\n\n" +
           "data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
           "data: [DONE]\n\n"
    // 验证重放功能
}
```

#### 测试案例 1.2: ReplayOpenAIStreamChatResponse - DeepSeek Reasoner 推理内容
**目标**: 验证支持 DeepSeek Reasoner 的 `reasoning_content` 字段
**输入**: 包含 `reasoning_content` 和 `content` 的 SSE 数据
**预期输出**:
- 正确解析推理内容 (`reasoning_content`)
- 正确解析回答文本 (`content`)
- 生成包含思考过程的完整消息
- 思考内容出现在回答文本之前

**测试代码参考**: `TestReplayOpenAIStreamChatResponse_reasoningAndText_deepseekReasoner`

#### 测试案例 1.3: ReplayOpenAINonStreamChatResponse - DeepSeek Reasoner 非流式
**目标**: 验证非流式响应中 DeepSeek Reasoner 的推理内容支持
**输入**: 包含 `reasoning_content` 和 `content` 的 JSON 响应
**预期输出**:
- 正确解析 `reasoning_content` 字段
- 正确解析 `content` 字段
- 生成包含思考过程的合成流事件
- 思考内容出现在回答文本之前

**测试代码参考**: `TestReplayOpenAINonStreamChatResponse_reasoningAndText_deepseekReasoner`

#### 测试案例 1.4: OpenAI 流式适配器 - 工具调用
**目标**: 验证工具调用的参数解析
**输入**: 包含工具调用的 SSE 数据，参数为 JSON 对象
**预期输出**:
- 正确解析工具名称 (`Bash`)
- 正确解析参数 (`command: "pwd"`)
- 生成 `input_json_delta` 事件
- 工具块正确构建

**测试代码参考**: `TestOpenAIStreamAdapter_toolCalls_argumentsAsObject`

### 测试组 2: gou-demo 流式工具渲染

#### 测试案例 2.1: 工具摘要延迟显示
**目标**: 验证 `GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS` 控制摘要行合并延迟
**输入**: 流式工具调用，设置 `GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS=1000`
**预期输出**:
- 助手消息首次出现后，先显示完整 Search/Read 行
- 到达 1000ms 后切换为合并摘要行
- 非阻塞渲染，UI 保持响应

#### 测试案例 2.2: 工具摘要延迟 - 关闭延迟
**目标**: 验证延迟关闭行为
**输入**: 设置 `GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS=0`
**预期输出**:
- 立即显示合并摘要行
- 不经过“先展示完整行”的延迟阶段

#### 测试案例 2.3: 工具摘要延迟 - 无效值
**目标**: 验证无效配置回退行为
**输入**: 设置非法值或负值
**预期输出**:
- 非法值或负值按关闭延迟处理
- 渲染逻辑不报错

#### 测试案例 2.4: 移除 time.Sleep 验证
**目标**: 验证代码中已移除阻塞的 `time.Sleep` 调用
**检查点**:
- `message_viewport_pane.go` 中无 `time.Sleep(2000 * time.Millisecond)` 调用
- 渲染逻辑完全基于时间计算
- UI 线程不被阻塞

### 测试组 3: 集成测试

#### 测试案例 3.1: 端到端 OpenAI 流式重放
**目标**: 验证完整的 OpenAI SSE 重放流程
**输入**: 完整的 OpenAI SSE 对话记录
**预期输出**:
- `ReplayOpenAIStreamChatResponse` 成功处理 SSE
- 生成正确的 Anthropic 流事件序列
- `assistantStreamAccumulator` 正确累积事件
- 最终 `AssistantWire` 包含完整对话内容

#### 测试案例 3.2: 混合内容类型处理
**目标**: 验证混合内容类型（文本、推理、工具）的处理
**输入**: 包含文本、推理内容和工具调用的 SSE 数据
**预期输出**:
- 正确区分和处理不同类型的内容块
- 推理内容正确转换为思考块
- 工具调用正确解析和渲染
- 内容顺序保持正确

#### 测试案例 3.3: 性能测试 - 非阻塞渲染
**目标**: 验证非阻塞渲染的性能
**测试方法**:
- 模拟多个并发工具调用
- 测量 UI 响应时间
- 验证无明显的渲染卡顿
- 检查内存使用情况

---

## 测试环境配置

### 环境变量
```bash
# OpenAI 适配器测试
export MODEL=deepseek-reasoner

# gou-demo 工具摘要延迟测试
export GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS=1000
export GOU_DEMO_BUBBLES_VIEWPORT=1
```

### 测试数据
测试数据应包含：
1. 纯文本 SSE 响应
2. 包含推理内容的 SSE 响应
3. 包含工具调用的 SSE 响应
4. 混合内容类型的 SSE 响应
5. 非流式 JSON 响应

---

## 预期行为

### 成功标准
1. **正确性**: 所有内容正确解析和渲染
2. **兼容性**: 支持 DeepSeek Reasoner 的推理内容
3. **性能**: 非阻塞渲染，UI 保持响应
4. **健壮性**: 处理边界情况和错误输入

### 错误处理
1. **无效 SSE**: 返回适当的错误
2. **无效 JSON**: 优雅降级或错误提示
3. **缺失字段**: 使用默认值或跳过
4. **环境变量错误**: 使用安全的默认值

---

## 测试工具和框架

### 单元测试
- 使用 Go 标准 `testing` 包
- 测试文件: `openai_stream_adapt_test.go`
- 辅助函数: `eventTypes`, 验证工具

### 集成测试
- 端到端流程测试
- 性能基准测试
- UI 响应性测试

### 手动测试
- gou-demo TUI 交互测试
- 视觉验证渲染效果
- 键盘交互测试

---

## 备注

1. **向后兼容性**: 新功能不应破坏现有测试
2. **性能影响**: 非阻塞渲染不应显著增加 CPU 使用
3. **内存使用**: 流式处理应保持合理的内存占用
4. **代码覆盖率**: 目标覆盖所有新增代码路径

---

**文档版本**: 1.0  
**最后更新**: 2026-04-17  
**相关提交**: 0a1060a, 2510dd2, 68b1798