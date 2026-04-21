package hookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

const hookEventUserPromptSubmit = "UserPromptSubmit"

type userPromptSubmitHookInput struct {
	BaseHookInput
	Prompt string `json:"prompt"`
}

// RunUserPromptSubmitHooks runs UserPromptSubmit **command** hooks (TS executeUserPromptSubmitHooks → executeHooks command branch)
// and returns [types.AggregatedHookResult] slices suitable for [processuserinput.ProcessUserInputParams.ExecuteUserPromptSubmitHooks].
//
// 这个函数执行用户提示提交钩子，处理以下流程：
// 1. 检查钩子是否被禁用（通过环境变量或策略）
// 2. 构建钩子输入 JSON，包含基本信息和用户提示
// 3. 从钩子表中匹配当前事件（UserPromptSubmit）的钩子
// 4. 并行执行所有匹配的钩子命令
// 5. 解析每个钩子的输出，进行验证和聚合
// 6. 返回聚合结果，供 processUserInput 处理
//
// 参数说明：
//   - ctx: 上下文，用于超时控制和取消
//   - table: 钩子表，从用户设置文件加载的钩子配置
//   - workDir: 工作目录，钩子命令执行的工作目录
//   - base: 基本钩子输入，包含会话ID、权限模式、代理信息等
//   - prompt: 用户提交的提示文本，用于钩子匹配和输入
//   - batchTimeoutMs: 批处理超时时间（毫秒），控制整个钩子执行的超时
//
// 返回值：
//   - []types.AggregatedHookResult: 聚合的钩子结果切片，每个结果包含：
//     - BlockingError: 阻塞错误，阻止继续处理
//     - PreventContinuation: 是否阻止继续
//     - PermissionBehavior: 权限行为（allow/deny/ask）
//     - AdditionalContexts: 附加上下文信息
//     - Message: 钩子生成的消息
//   - error: 执行过程中的错误，如JSON解析失败、钩子加载失败等
//
// 钩子输出格式支持：
//   - JSON格式（推荐）：包含结构化决策信息
//     {
//       "continue": false,
//       "decision": "block",
//       "reason": "违反政策",
//       "systemMessage": "操作被阻止"
//     }
//   - 简单文本格式：非JSON输出，根据退出码处理
//     - 退出码0：成功，输出作为成功消息
//     - 退出码2：阻塞错误，阻止继续
//     - 其他退出码：非阻塞错误，记录但不阻止
//
// 钩子匹配规则：
//   - 根据 hook_event_name = "UserPromptSubmit" 选择钩子组
//   - 使用 DeriveMatchQuery 从 prompt 生成匹配查询
//   - 根据 matcher 正则表达式过滤钩子
//   - 支持多个钩子并行执行
//
// 与TypeScript的奇偶性：
//   - 模仿 src/utils/hooks.ts 中的 executeUserPromptSubmitHooks 函数
//   - 支持相同的输入输出格式
//   - 保持相同的匹配和执行语义
func RunUserPromptSubmitHooks(ctx context.Context, table HooksTable, workDir string, base BaseHookInput, prompt string, batchTimeoutMs int) ([]types.AggregatedHookResult, error) {
	// 1. 检查钩子是否被全局禁用
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}
	
	// 2. 构建完整的钩子输入
	base.HookEventName = hookEventUserPromptSubmit
	in := userPromptSubmitHookInput{BaseHookInput: base, Prompt: prompt}
	jsonIn, err := marshalHookInput(in)
	if err != nil {
		return nil, err
	}
	
	// 3. 将JSON输入转换为map，用于钩子匹配
	var hookInput map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonIn)), &hookInput); err != nil {
		return nil, err
	}
	
	// 4. 获取匹配的钩子命令，如果没有匹配的钩子则直接返回
	if len(CommandHooksForHookInput(table, hookInput)) == 0 {
		return nil, nil
	}
	
	// 5. 确保工作目录有效
	wd := trimOrDot(workDir)
	
	// 6. 并行执行所有匹配的钩子命令
	results := ExecuteCommandHooksOutsideREPLParallel(OutsideReplCommandParams{
		Ctx:       ctx,
		WorkDir:   wd,
		Hooks:     table,
		JSONInput: jsonIn,
		TimeoutMs: batchTimeoutMs,
	})
	
	// 7. 为这批钩子生成唯一的工具使用ID
	toolUseID := randomUUID()
	
	// 8. 聚合所有钩子的执行结果
	var agg []types.AggregatedHookResult
	for _, r := range results {
		// 对每个钩子结果进行解析和聚合
		agg = append(agg, userPromptSubmitAggregates(r, toolUseID, hookEventUserPromptSubmit, r.Command)...)
	}
	
	// 9. 返回聚合结果
	return agg, nil
}

func userPromptSubmitAggregates(r OutsideReplCommandResult, toolUseID, hookEvent, hookName string) []types.AggregatedHookResult {
	stdout := strings.TrimSpace(r.Stdout)
	trimmed := strings.TrimSpace(stdout)

	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return userPromptSubmitNonJSONPath(r, toolUseID, hookEvent, hookName)
	}

	var asyncProbe struct {
		Async *bool `json:"async"`
	}
	if err := json.Unmarshal([]byte(trimmed), &asyncProbe); err != nil {
		return userPromptSubmitNonJSONPath(r, toolUseID, hookEvent, hookName)
	}
	if asyncProbe.Async != nil && *asyncProbe.Async {
		return nil
	}

	parsed, err := parseSyncHookStdoutJSON(trimmed)
	if err != nil {
		return userPromptSubmitValidationError(r, toolUseID, hookEvent, hookName, err.Error())
	}
	top := parsedSyncHookToLegacyTop(parsed)

	if len(top.HookSpecificOutput) > 0 {
		var hso struct {
			HookEventName string `json:"hookEventName"`
		}
		_ = json.Unmarshal(top.HookSpecificOutput, &hso)
		if hso.HookEventName != "" && hso.HookEventName != hookEventUserPromptSubmit {
			return userPromptSubmitValidationError(r, toolUseID, hookEvent, hookName,
				fmt.Sprintf("Hook returned incorrect event name: expected %q but got %q. Full stdout: %s", hookEventUserPromptSubmit, hso.HookEventName, trimmed))
		}
	}

	var out []types.AggregatedHookResult

	if top.Continue != nil && !*top.Continue {
		item := types.AggregatedHookResult{PreventContinuation: boolPtr(true)}
		if top.StopReason != nil && strings.TrimSpace(*top.StopReason) != "" {
			sr := strings.TrimSpace(*top.StopReason)
			item.StopReason = &sr
		}
		out = append(out, item)
	}

	switch strings.TrimSpace(top.Decision) {
	case "block":
		reason := strings.TrimSpace(top.Reason)
		if reason == "" {
			reason = "Blocked by hook"
		}
		out = append(out, types.AggregatedHookResult{
			BlockingError: &types.HookBlockingError{BlockingError: reason, Command: r.Command},
		})
	case "approve":
		allow := "allow"
		item := types.AggregatedHookResult{PermissionBehavior: &allow}
		if strings.TrimSpace(top.Reason) != "" {
			rs := strings.TrimSpace(top.Reason)
			item.HookPermissionDecisionReason = &rs
		}
		out = append(out, item)
	}

	if strings.TrimSpace(top.SystemMessage) != "" {
		msg, err := serializedHookSystemMessage(toolUseID, hookName, hookEvent, strings.TrimSpace(top.SystemMessage))
		if err == nil && len(msg) > 0 {
			out = append(out, types.AggregatedHookResult{Message: msg})
		}
	}

	if len(top.HookSpecificOutput) > 0 {
		var hso struct {
			AdditionalContext string `json:"additionalContext"`
		}
		if err := json.Unmarshal(top.HookSpecificOutput, &hso); err == nil && strings.TrimSpace(hso.AdditionalContext) != "" {
			out = append(out, types.AggregatedHookResult{
				AdditionalContexts: []string{hso.AdditionalContext},
			})
		}
	}

	msg, err := serializedHookProcessJSONMessage(r, toolUseID, hookEvent, hookName, top)
	if err == nil && len(msg) > 0 {
		out = append(out, types.AggregatedHookResult{Message: msg})
	}

	return out
}

type syncUserPromptSubmitJSON struct {
	Continue           *bool           `json:"continue"`
	StopReason         *string         `json:"stopReason"`
	Decision           string          `json:"decision"`
	Reason             string          `json:"reason"`
	SystemMessage      string          `json:"systemMessage"`
	SuppressOutput     *bool           `json:"suppressOutput"`
	HookSpecificOutput json.RawMessage `json:"hookSpecificOutput"`
}

func userPromptSubmitNonJSONPath(r OutsideReplCommandResult, toolUseID, hookEvent, hookName string) []types.AggregatedHookResult {
	stdout := strings.TrimSpace(r.Stdout)
	stderr := strings.TrimSpace(r.Stderr)
	exit := r.ExitCode

	if r.Succeeded && exit == 0 {
		if stdout == "" {
			return nil
		}
		msg, err := serializedHookSuccess(toolUseID, hookName, hookEvent, stdout, r.Stdout, r.Stderr, exit, r.Command, r.DurationMs)
		if err != nil || len(msg) == 0 {
			return nil
		}
		return []types.AggregatedHookResult{{Message: msg}}
	}

	if exit == 2 {
		s := stderr
		if s == "" {
			s = "No stderr output"
		}
		blocking := fmt.Sprintf("[%s]: %s", r.Command, s)
		return []types.AggregatedHookResult{{
			BlockingError: &types.HookBlockingError{BlockingError: blocking, Command: r.Command},
		}}
	}

	errLine := stderr
	if errLine == "" {
		errLine = "No stderr output"
	}
	detail := fmt.Sprintf("Failed with non-blocking status code: %s", errLine)
	msg, err := serializedHookNonBlockingError(toolUseID, hookName, hookEvent, detail, r.Stdout, exit, r.Command, r.DurationMs)
	if err != nil || len(msg) == 0 {
		return nil
	}
	return []types.AggregatedHookResult{{Message: msg}}
}

func userPromptSubmitValidationError(r OutsideReplCommandResult, toolUseID, hookEvent, hookName, detail string) []types.AggregatedHookResult {
	msg, err := serializedHookNonBlockingError(toolUseID, hookName, hookEvent, "JSON validation failed: "+detail, r.Stdout, 1, r.Command, r.DurationMs)
	if err != nil || len(msg) == 0 {
		return nil
	}
	return []types.AggregatedHookResult{{Message: msg}}
}

func serializedHookProcessJSONMessage(r OutsideReplCommandResult, toolUseID, hookEvent, hookName string, top syncUserPromptSubmitJSON) (json.RawMessage, error) {
	if strings.TrimSpace(top.Decision) == "block" {
		reason := strings.TrimSpace(top.Reason)
		if reason == "" {
			reason = "Blocked by hook"
		}
		att := map[string]any{
			"type": "hook_blocking_error",
			"blockingError": map[string]any{
				"blockingError": reason,
				"command":       r.Command,
			},
			"hookName":  hookName,
			"hookEvent": hookEvent,
		}
		return marshalAttachmentMessage(toolUseID, att)
	}
	content := ""
	if suppressOutputFalse(top.SuppressOutput) && strings.TrimSpace(r.Stdout) != "" && r.ExitCode == 0 && r.Succeeded {
		content = ""
	}
	return serializedHookSuccess(toolUseID, hookName, hookEvent, content, r.Stdout, r.Stderr, r.ExitCode, r.Command, r.DurationMs)
}

func suppressOutputFalse(p *bool) bool {
	return p == nil || !*p
}

func marshalAttachmentMessage(toolUseID string, attachment map[string]any) (json.RawMessage, error) {
	attachment["toolUseID"] = toolUseID
	rawAtt, err := json.Marshal(attachment)
	if err != nil {
		return nil, err
	}
	msg := map[string]any{
		"type":       string(types.MessageTypeAttachment),
		"uuid":       randomUUID(),
		"attachment": json.RawMessage(rawAtt),
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func serializedHookSuccess(toolUseID, hookName, hookEvent, content, stdout, stderr string, exitCode int, command string, durationMs int64) (json.RawMessage, error) {
	att := map[string]any{
		"type":       "hook_success",
		"content":    content,
		"hookName":   hookName,
		"hookEvent":  hookEvent,
		"stdout":     stdout,
		"stderr":     stderr,
		"exitCode":   exitCode,
		"command":    command,
		"durationMs": durationMs,
	}
	return marshalAttachmentMessage(toolUseID, att)
}

func serializedHookNonBlockingError(toolUseID, hookName, hookEvent, stderr, stdout string, exitCode int, command string, durationMs int64) (json.RawMessage, error) {
	att := map[string]any{
		"type":       "hook_non_blocking_error",
		"hookName":   hookName,
		"stderr":     stderr,
		"stdout":     stdout,
		"exitCode":   exitCode,
		"hookEvent":  hookEvent,
		"command":    command,
		"durationMs": durationMs,
	}
	return marshalAttachmentMessage(toolUseID, att)
}

func serializedHookSystemMessage(toolUseID, hookName, hookEvent, content string) (json.RawMessage, error) {
	att := map[string]any{
		"type":      "hook_system_message",
		"content":   content,
		"hookName":  hookName,
		"hookEvent": hookEvent,
	}
	return marshalAttachmentMessage(toolUseID, att)
}

func boolPtr(b bool) *bool { return &b }
