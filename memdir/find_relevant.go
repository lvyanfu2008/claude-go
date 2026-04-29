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
	"goc/modelenv"

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
	Model        string      `json:"model"`
	MaxTokens    int         `json:"max_tokens"`
	Messages     []msgPart   `json:"messages"`
	System       string      `json:"system,omitempty"`
	Stream       bool        `json:"stream,omitempty"`
	OutputFormat any         `json:"output_format,omitempty"`
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

type selectMemoriesOutput struct {
	SelectedMemories []string `json:"selected_memories"`
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

// selectRelevantMemories calls Sonnet to pick up to 5 most relevant memory filenames.
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

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_AUTH_TOKEN")
	}
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")

	sonnetModel := modelenv.ResolveWithFallback("claude-sonnet-4-20250514")

	req := messageRequest{
		Model:     sonnetModel,
		MaxTokens: 256,
		System:    selectMemoriesSystemPrompt,
		Messages: []msgPart{
			{
				Role:    "user",
				Content: fmt.Sprintf("Query: %s\n\nAvailable memories:\n%s%s", query, manifest, toolsSection),
			},
		},
		OutputFormat: map[string]any{
			"type": "json_schema",
			"schema": map[string]any{
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
			},
		},
	}

	resp, err := postMessage(ctx, apiKey, baseURL, req, []string{"structured-outputs-2025-12-15"})
	if err != nil {
		return nil, err
	}

	blocks, err := parseContentBlocks(resp.Content)
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

	var parsed selectMemoriesOutput
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("parse selected_memories JSON: %w", err)
	}

	var selected []string
	for _, f := range parsed.SelectedMemories {
		if validFilenames[f] {
			selected = append(selected, f)
		}
	}
	return selected, nil
}

// FindRelevantMemories mirrors src/memdir/findRelevantMemories.ts.
// It scans the memory directory, calls Sonnet to select up to 5 most relevant
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
