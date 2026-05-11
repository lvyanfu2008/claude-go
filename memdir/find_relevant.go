package memdir

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"goc/anthropicmessages"
	"goc/ccb-engine/apilog"
	"goc/tstenv"

	"gopkg.in/yaml.v3"
)

// MemoryHeader mirrors src/memdir/memoryScan.ts MemoryHeader.
type MemoryHeader struct {
	Filename    string
	FilePath    string
	MtimeMs     int64
	Description string
	Type        string
}

// RelevantMemory mirrors src/memdir/findRelevantMemories.ts RelevantMemory.
type RelevantMemory struct {
	Path    string
	MtimeMs int64
}

// frontmatterRE matches YAML frontmatter delimited by ---.
var frontmatterRE = regexp.MustCompile(`(?s)^---\s*\n([\s\S]*?)---`)

const (
	maxMemoryFiles       = 200
	frontmatterReadBytes = 8192 // enough for 30 lines of YAML frontmatter
)

const selectMemoriesSystemPrompt = `You are selecting memories that will be useful to Claude Code as it processes a user's query. You will be given the user's query and a list of available memory files with their filenames and descriptions.

Return a list of filenames for the memories that will clearly be useful to Claude Code as it processes the user's query (up to 5). Only include memories that you are certain will be helpful based on their name and description.
- If you are unsure if a memory will be useful in processing the user's query, then do not include it in your list. Be selective and discerning.
- If there are no memories in the list that would clearly be useful, feel free to return an empty list.
- If a list of recently-used tools is provided, do not select memories that are usage reference or API documentation for those tools (Claude Code is already exercising them). DO still select memories containing warnings, gotchas, or known issues about those tools — active use is exactly when those matter.
`

// parseFrontmatterYAML extracts YAML frontmatter from markdown content.
func parseFrontmatterYAML(content string) map[string]interface{} {
	m := frontmatterRE.FindStringSubmatch(content)
	if m == nil {
		return map[string]interface{}{}
	}
	raw := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(m[1]), &raw); err != nil {
		return map[string]interface{}{}
	}
	return raw
}

// scanMemoryFiles scans a memory directory for .md files, reads their frontmatter,
// and returns a header list sorted newest-first (capped at maxMemoryFiles).
func ScanMemoryFiles(memoryDir string) []MemoryHeader {
	var headers []MemoryHeader

	_ = filepath.WalkDir(memoryDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") || d.Name() == "MEMORY.md" {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		buf := make([]byte, frontmatterReadBytes)
		n, _ := f.Read(buf)

		fm := parseFrontmatterYAML(string(buf[:n]))
		headers = append(headers, MemoryHeader{
			Filename:    d.Name(),
			FilePath:    path,
			MtimeMs:     info.ModTime().UnixMilli(),
			Description: frontmatterString(fm, "description"),
			Type:        frontmatterString(fm, "type"),
		})
		return nil
	})

	sort.Slice(headers, func(i, j int) bool {
		return headers[i].MtimeMs > headers[j].MtimeMs
	})
	if len(headers) > maxMemoryFiles {
		headers = headers[:maxMemoryFiles]
	}
	return headers
}

func frontmatterString(fm map[string]interface{}, key string) string {
	v, ok := fm[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// formatMemoryManifest formats memory headers as a text manifest for the selector prompt.
func FormatMemoryManifest(memories []MemoryHeader) string {
	lines := make([]string, 0, len(memories))
	for _, m := range memories {
		tag := ""
		if m.Type != "" {
			tag = "[" + m.Type + "] "
		}
		ts := time.UnixMilli(m.MtimeMs).UTC().Format(time.RFC3339)
		if m.Description != "" {
			lines = append(lines, fmt.Sprintf("- %s%s (%s): %s", tag, m.Filename, ts, m.Description))
		} else {
			lines = append(lines, fmt.Sprintf("- %s%s (%s)", tag, m.Filename, ts))
		}
	}
	return strings.Join(lines, "\n")
}

// ---- minimal non-streaming Messages API call (avoids import cycle through internal/anthropic) ----

const defaultAnthropicBaseURL = "https://api.anthropic.com"
const anthropicAPIVersion = "2023-06-01"

type messageRequest struct {
	Model        string    `json:"model"`
	MaxTokens    int       `json:"max_tokens"`
	Messages     []msgPart `json:"messages"`
	System       string    `json:"system,omitempty"`
	Stream       bool      `json:"stream,omitempty"`
	OutputFormat any       `json:"output_format,omitempty"`
}

type msgPart struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageResponse struct {
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func postMessage(ctx context.Context, apiKey, baseURL string, req messageRequest, betaHeaders []string) (*messageResponse, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("memdir: missing API key")
	}
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 256
	}
	req.Stream = false

	body, err := jsonMarshalNoEscape(req)
	if err != nil {
		return nil, err
	}

	msgURL := anthropicmessages.MessagesAPIURL(baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, msgURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)
	httpReq.Header.Set("content-type", "application/json")
	if len(betaHeaders) > 0 {
		httpReq.Header.Set("anthropic-beta", strings.Join(betaHeaders, ","))
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("anthropic API %s: %s", resp.Status, truncateStr(string(respBody), 800))
	}

	var out messageResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// modelSupportsStructuredOutputs mirrors src/utils/betas.ts modelSupportsStructuredOutputs.
// Structured outputs beta header is only safe for firstParty and Foundry providers
// (not Bedrock, Vertex, OpenAI/DeepSeek, etc.).
func modelSupportsStructuredOutputs(model string) bool {
	provider := tstenv.GetAPIProvider()
	if provider != tstenv.FirstParty && provider != tstenv.Foundry {
		return false
	}
	canonical := strings.ToLower(model)
	return strings.Contains(canonical, "claude-sonnet-4-6") ||
		strings.Contains(canonical, "claude-sonnet-4-5") ||
		strings.Contains(canonical, "claude-opus-4-1") ||
		strings.Contains(canonical, "claude-opus-4-5") ||
		strings.Contains(canonical, "claude-opus-4-6") ||
		strings.Contains(canonical, "claude-haiku-4-5")
}

func jsonMarshalNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func parseContentBlocks(raw json.RawMessage) ([]contentBlock, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// selectMemoriesOutputSchema is the JSON schema embedded in the prompt (Anthropic path
// uses native json_schema output_format; OpenAI/DeepSeek path uses json_object and embeds
// this schema as a text hint so the model knows the expected field names).
var selectMemoriesOutputSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"selected_memories": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "string",
			},
		},
	},
	"required":             []string{"selected_memories"},
	"additionalProperties": false,
}

// selectMemoriesJSONSchemaText is the JSON schema serialized as a string for embedding
// in the OpenAI/DeepSeek prompt (used by schemaToPromptHint equivalent).
func selectMemoriesJSONSchemaText() string {
	b, _ := json.Marshal(selectMemoriesOutputSchema)
	return string(b)
}

// getDefaultHaikuModelForMemdir mirrors TS getDefaultHaikuModel()
// (src/utils/model/model.ts:164-181).
func getDefaultHaikuModelForMemdir() string {
	provider := tstenv.GetAPIProvider()
	// For OpenAI/DeepSeek provider, check OPENAI_DEFAULT_HAIKU_MODEL first
	if provider == tstenv.OpenAI {
		if v := strings.TrimSpace(os.Getenv("OPENAI_DEFAULT_HAIKU_MODEL")); v != "" {
			return v
		}
	}
	// Anthropic-specific override (for first-party and other providers)
	if v := strings.TrimSpace(os.Getenv("ANTHROPIC_DEFAULT_HAIKU_MODEL")); v != "" {
		return v
	}
	// Haiku 4.5 is available on all platforms
	return "claude-haiku-4-5-20251001"
}

// selectRelevantMemories mirrors TS selectRelevantMemories (sideQuery.ts) called by selectRelevantMemories
// (findRelevantMemories.ts:77-157). Routes through Anthropic Messages API for
// firstParty/foundry, OpenAI Chat Completions for OpenAI/DeepSeek. Other providers
// (Bedrock/Vertex/Gemini/Grok) return nil (no selectRelevantMemories path).
func selectRelevantMemories(
	ctx context.Context,
	query string,
	memories []MemoryHeader,
	recentTools []string,
) ([]string, error) {
	validFilenames := make(map[string]bool, len(memories))
	for _, m := range memories {
		validFilenames[m.Filename] = true
	}

	manifest := FormatMemoryManifest(memories)

	toolsSection := ""
	if len(recentTools) > 0 {
		toolsSection = "\n\nRecently used tools: " + strings.Join(recentTools, ", ")
	}

	// DeepSeek models do not support the Anthropic /v1/messages endpoint.
	// When the resolved Haiku model is DeepSeek, route through OpenAI Chat
	// Completions instead, regardless of API provider.
	haikuModel := getDefaultHaikuModelForMemdir()
	if tstenv.IsDeepSeekModel(haikuModel) {
		return sideQueryOpenAI(ctx, query, manifest, toolsSection, validFilenames)
	}

	provider := tstenv.GetAPIProvider()
	switch provider {
	case tstenv.FirstParty, tstenv.Foundry:
		return sideQueryAnthropic(ctx, query, manifest, toolsSection, validFilenames)
	case tstenv.OpenAI:
		return sideQueryOpenAI(ctx, query, manifest, toolsSection, validFilenames)
	default:
		// Bedrock, Vertex, Gemini, Grok — no sideQuery path; mirror TS
		// sideQuery failure → catch → return [].
		return nil, nil
	}
}

// sideQueryAnthropic uses the Anthropic Messages API with native
// json_schema output_format (structured outputs beta when supported).
func sideQueryAnthropic(
	ctx context.Context,
	query, manifest, toolsSection string,
	validFilenames map[string]bool,
) ([]string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_AUTH_TOKEN")
	}
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")

	haikuModel := getDefaultHaikuModelForMemdir()

	req := messageRequest{
		Model:     haikuModel,
		MaxTokens: 256,
		System:    selectMemoriesSystemPrompt,
		Messages: []msgPart{
			{
				Role:    "user",
				Content: fmt.Sprintf("Query: %s\n\nAvailable memories:\n%s%s", query, manifest, toolsSection),
			},
		},
		OutputFormat: map[string]any{
			"type":   "json_schema",
			"schema": selectMemoriesOutputSchema,
		},
	}

	var betaHeaders []string
	if modelSupportsStructuredOutputs(haikuModel) {
		betaHeaders = []string{"structured-outputs-2025-12-15"}
	}
	resp, err := postMessage(ctx, apiKey, baseURL, req, betaHeaders)
	if err != nil {
		return nil, err
	}

	return parseSelectMemoriesResponse(resp.Content, validFilenames)
}

// openAIChatCompletionRequest is the minimal OpenAI Chat Completions request body.
type openAIChatCompletionRequest struct {
	Model          string           `json:"model"`
	Messages       []openAIChatMsg  `json:"messages"`
	MaxTokens      int              `json:"max_tokens"`
	ResponseFormat *json.RawMessage `json:"response_format,omitempty"`
}

type openAIChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// sideQueryOpenAI uses OpenAI Chat Completions API (non-streaming)
// with response_format: json_object. The JSON schema is embedded as a text hint
// in the system prompt so the model knows which key names to use
// (e.g. "selected_memories" vs "memory_files").
//
// This path is DeepSeek-compatible: DeepSeek and many other OpenAI-compatible
// providers do not support the json_schema response_format type, only json_object.
// DeepSeek also requires the word "json" in the prompt when using json_object,
// which the schema text hint satisfies.
func sideQueryOpenAI(
	ctx context.Context,
	query, manifest, toolsSection string,
	validFilenames map[string]bool,
) ([]string, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("memdir: OPENAI_API_KEY missing for OpenAI provider")
	}
	baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Resolve Haiku model to OpenAI equivalent.
	haikuModel := getDefaultHaikuModelForMemdir()
	openaiModel := resolveOpenAIModelInline(haikuModel)

	// Build the JSON schema text hint for DeepSeek compatibility.
	// Mirrors TS schemaToPromptHint (sideQuery.ts:103-108).
	schemaText := selectMemoriesJSONSchemaText()
	formatHint := fmt.Sprintf("Respond in JSON format. The response must conform to this JSON schema:\n%s", schemaText)

	// Build system prompt with format hint appended.
	// The format hint includes the word "json" which DeepSeek requires when using
	// response_format: json_object.
	systemContent := selectMemoriesSystemPrompt + "\n" + formatHint

	rf := json.RawMessage(`{"type":"json_object"}`)
	reqBody := openAIChatCompletionRequest{
		Model: openaiModel,
		Messages: []openAIChatMsg{
			{Role: "system", Content: systemContent},
			{
				Role:    "user",
				Content: fmt.Sprintf("Query: %s\n\nAvailable memories:\n%s%s", query, manifest, toolsSection),
			},
		},
		MaxTokens:      256,
		ResponseFormat: &rf,
	}

	body, err := jsonMarshalNoEscape(reqBody)
	if err != nil {
		return nil, err
	}
	apilog.LogRequestBody("memdir sideQueryOpenAI", body)

	url := baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("authorization", "Bearer "+apiKey)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	apilog.LogResponseBody("memdir sideQueryOpenAI", respBody)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai chat %s: %s", resp.Status, truncateStr(string(respBody), 800))
	}

	var chatResp openAIChatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}
	if chatResp.Error != nil && strings.TrimSpace(chatResp.Error.Message) != "" {
		return nil, fmt.Errorf("openai api error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("openai: empty choices")
	}

	text := chatResp.Choices[0].Message.Content
	// Wrap plain text string as content blocks array matching Anthropic format
	// so parseContentBlocks can decode it. Mirrors TS sideQuery.ts:323-326.
	wrapped, err := json.Marshal([]contentBlock{{Type: "text", Text: text}})
	if err != nil {
		return nil, fmt.Errorf("marshal content blocks: %w", err)
	}
	return parseSelectMemoriesResponse(wrapped, validFilenames)
}

// resolveOpenAIModelInline is a minimal inline version of ResolveOpenAIModel
// (query/openai_model_resolve.go) for use in memdir without importing the query package.
func resolveOpenAIModelInline(anthropicModel string) string {
	if v := strings.TrimSpace(os.Getenv("OPENAI_DEFAULT_HAIKU_MODEL")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("ANTHROPIC_DEFAULT_HAIKU_MODEL")); v != "" {
		return v
	}
	clean := strings.TrimSuffix(strings.TrimSpace(anthropicModel), "[1m]")
	low := strings.ToLower(clean)
	if strings.Contains(low, "haiku") {
		return "gpt-4o-mini"
	}
	if strings.Contains(low, "sonnet") {
		return "gpt-4o"
	}
	if strings.Contains(low, "opus") {
		return "o3"
	}
	switch clean {
	case "claude-haiku-4-5-20251001", "claude-3-5-haiku-20241022":
		return "gpt-4o-mini"
	case "claude-sonnet-4-20250514", "claude-sonnet-4-5-20250929",
		"claude-sonnet-4-6", "claude-3-7-sonnet-20250219",
		"claude-3-5-sonnet-20241022":
		return "gpt-4o"
	case "claude-opus-4-20250514", "claude-opus-4-1-20250805",
		"claude-opus-4-5-20251101", "claude-opus-4-6":
		return "o3"
	}
	if clean == "" {
		return "gpt-4o"
	}
	return clean
}

// parseSelectMemoriesResponse parses the LLM response content into selected filenames.
// Mirrors TS findRelevantMemories.ts:129-146 — tries selected_memories first,
// then falls back to memory_files.
func parseSelectMemoriesResponse(raw json.RawMessage, validFilenames map[string]bool) ([]string, error) {
	blocks, err := parseContentBlocks(raw)
	if err != nil {
		return nil, fmt.Errorf("parse content blocks: %w", err)
	}

	var textParts []string
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			textParts = append(textParts, block.Text)
		}
	}
	text := strings.Join(textParts, "")

	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	// Try selected_memories first, then memory_files (TS fallback order).
	var filenames []string
	if arr, ok := parsed["selected_memories"].([]any); ok {
		for _, f := range arr {
			if s, ok := f.(string); ok {
				filenames = append(filenames, s)
			}
		}
	} else if arr, ok := parsed["memory_files"].([]any); ok {
		for _, f := range arr {
			if s, ok := f.(string); ok {
				filenames = append(filenames, s)
			}
		}
	}

	var selected []string
	for _, f := range filenames {
		if validFilenames[f] {
			selected = append(selected, f)
		}
	}
	return selected, nil
}

// FindRelevantMemories mirrors src/memdir/findRelevantMemories.ts.
// It scans the memory directory, calls Haiku to select up to 5 most relevant
// memory files for the given query, and returns their absolute paths + mtimes.
//
// alreadySurfaced filters out paths shown in prior turns so the selector
// spends its 5-slot budget on fresh candidates.
func FindRelevantMemories(
	ctx context.Context,
	query string,
	memoryDir string,
	recentTools []string,
	alreadySurfaced map[string]bool,
) []RelevantMemory {
	memories := ScanMemoryFiles(memoryDir)
	if alreadySurfaced != nil {
		filtered := memories[:0]
		for _, m := range memories {
			if !alreadySurfaced[m.FilePath] {
				filtered = append(filtered, m)
			}
		}
		memories = filtered
	}
	if len(memories) == 0 {
		return nil
	}

	selectedFilenames, err := selectRelevantMemories(ctx, query, memories, recentTools)
	if err != nil {
		return nil
	}

	byFilename := make(map[string]MemoryHeader, len(memories))
	for _, m := range memories {
		byFilename[m.Filename] = m
	}

	var result []RelevantMemory
	for _, filename := range selectedFilenames {
		if m, ok := byFilename[filename]; ok {
			result = append(result, RelevantMemory{
				Path:    m.FilePath,
				MtimeMs: m.MtimeMs,
			})
		}
	}
	return result
}
