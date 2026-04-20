# Gemma 模型集成指南

## 概述

已将 Google 的 Gemma 模型集成到 Claude Go 的 LLM 模型系统中。Gemma 通过 Vertex AI 提供服务，可以通过单一环境变量切换使用。

## 启用 Gemma 模型

有两种方式启用 Gemma 模型：

### 方式一：使用专用环境变量
```bash
export CLAUDE_CODE_USE_GEMMA=true
```

### 方式二：通过模型名称识别
```bash
export CCB_ENGINE_MODEL=gemma-7b
```

## 配置 Vertex AI

使用 Gemma 模型需要配置 Vertex AI 环境变量：

```bash
# 必需配置
export VERTEX_AI_PROJECT_ID="your-project-id"
export VERTEX_AI_LOCATION="us-central1"
export VERTEX_AI_ENDPOINT_ID="your-endpoint-id"

# 可选配置（使用默认值）
export VERTEX_AI_MODEL_NAME="gemma-7b"  # 默认为 gemma-7b
```

## 认证

Gemma 集成使用 Google Cloud 默认凭证。确保已设置正确的认证：

```bash
# 方式一：使用服务账号密钥
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"

# 方式二：使用 gcloud 认证
gcloud auth application-default login
```

## 使用示例

### 1. 基本使用
```bash
# 启用 Gemma
export CLAUDE_CODE_USE_GEMMA=true

# 配置 Vertex AI
export VERTEX_AI_PROJECT_ID="my-project"
export VERTEX_AI_LOCATION="us-central1"
export VERTEX_AI_ENDPOINT_ID="my-endpoint"

# 运行 Claude Go
go run ./cmd/gou-demo
```

### 2. 与其他模型切换
```bash
# 使用 Gemma
export CLAUDE_CODE_USE_GEMMA=true

# 切换回 Anthropic Claude
unset CLAUDE_CODE_USE_GEMMA
export ANTHROPIC_API_KEY="your-api-key"

# 切换回 OpenAI
unset CLAUDE_CODE_USE_GEMMA
export CLAUDE_CODE_USE_OPENAI=true
export OPENAI_API_KEY="your-api-key"
```

## 模型选择优先级

系统按以下优先级选择模型提供者：

1. **Gemma** - 如果 `CLAUDE_CODE_USE_GEMMA=true` 或 `CCB_ENGINE_MODEL` 包含 "gemma"
2. **OpenAI** - 如果 `CLAUDE_CODE_USE_OPENAI=true`
3. **Anthropic** - 默认提供者

## 技术实现

### 文件结构
```
ccb-engine/gemma/
├── client.go          # Gemma 客户端实现
conversation-runtime/query/
├── gemma_provider.go  # Gemma 模型提供者
├── query.go           # 修改了模型选择逻辑
```

### 核心组件

1. **Gemma 客户端** (`ccb-engine/gemma/client.go`)
   - 封装 Vertex AI API 调用
   - 提供 OpenAI 兼容的接口
   - 处理认证和请求转换

2. **Gemma 提供者** (`conversation-runtime/query/gemma_provider.go`)
   - 检查 Gemma 启用状态
   - 实现 `runGemmaStreamingParityModelLoop` 函数
   - 处理消息格式转换

3. **查询逻辑** (`conversation-runtime/query/query.go`)
   - 添加 Gemma 到模型选择逻辑
   - 在 streaming parity 路径中支持 Gemma

## 注意事项

1. **工具支持**: Gemma 支持工具调用，但需要确保工具定义与模型兼容
2. **流式响应**: 当前实现使用非流式调用，因为 Vertex AI rawPredict 端点不支持流式响应
3. **Token 计数**: 使用简单的估算方法，实际生产环境应使用正确的 tokenizer
4. **错误处理**: 确保 Vertex AI 端点已正确部署 Gemma 模型

## 故障排除

### 常见问题

1. **认证失败**
   ```
   Error: 获取凭证失败
   ```
   解决方案：确保已正确设置 Google Cloud 认证

2. **端点不可用**
   ```
   Error: Vertex AI 返回错误: 404
   ```
   解决方案：检查 `VERTEX_AI_ENDPOINT_ID` 是否正确

3. **模型不响应**
   ```
   Error: no predictions in response
   ```
   解决方案：检查端点是否已部署 Gemma 模型

### 调试模式

启用详细日志：
```bash
export GOU_DEMO_LOG=true
export CLAUDE_CODE_LOG_API_REQUEST_BODY=true
```

## 性能考虑

1. **延迟**: Vertex AI 调用可能比直接 API 调用有更高延迟
2. **成本**: 使用 Vertex AI 会产生相应的 Google Cloud 费用
3. **配额**: 注意 Vertex AI 的配额限制

## 未来改进

1. 支持流式响应
2. 添加更准确的 token 计数
3. 支持更多 Gemma 模型变体
4. 添加本地 Gemma 模型支持