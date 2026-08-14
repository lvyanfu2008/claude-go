package query

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"goc/anthropicmessages"
)

// DSML tool-call grammar (DeepSeek V3.2/V4). Real-world output is heavily
// mangled: tags span lines, values span lines, delimiter bars are dropped or
// doubled, attributes glue together (parametername=), and markers split across
// stream chunks. We therefore NORMALIZE the buffered text first (strip bars,
// collapse whitespace) and match the simplified structure on the normalized
// text. Collapsing whitespace inside string values is acceptable: shell
// commands treat newlines as spaces, and this only affects already-mangled
// output.
var dsmlWSRe = regexp.MustCompile(`\s+`)

func normalizeDSML(s string) string {
	s = strings.ReplaceAll(s, "｜", " ")
	s = strings.ReplaceAll(s, "|", " ")
	return dsmlWSRe.ReplaceAllString(s, " ")
}

// These match NORMALIZED text (see normalizeDSML), used ONLY for start-tag
// detection and the hold-back prefix check. Extraction operates on RAW text via
// the dsmlRaw* regexes below so parameter VALUES keep their pipes and other
// shell syntax intact.
var (
	// \A-anchored on normalized text: a block only takes over when the buffered
	// text BEGINS with the tag, so an ordinary prose prefix can never be misread
	// as a block.
	dsmlBlockStartRe = regexp.MustCompile(`\A<\s*DSML\s*tool_calls\s*>`)
)

// Raw-tolerant regexes: bars (| or ｜) are optional between segments and any
// stray whitespace around them is skipped, so real mangled output like
// "< | DSML |invoke" or "<｜DSML｜parameter>" still parses.
var (
	dsmlRawStartRe  = regexp.MustCompile(`<\s*[|｜]?\s*DSML\s*[|｜]?\s*tool_calls\s*[|｜]?\s*>`)
	dsmlRawEndRe    = regexp.MustCompile(`(?s)</\s*[|｜]?\s*DSML\s*[|｜]?\s*tool_calls\s*[|｜]?\s*>`)
	dsmlRawInvokeRe = regexp.MustCompile(`(?s)<\s*[|｜]?\s*DSML\s*[|｜]?\s*invoke\s*[|｜]?\s*name="([^"]*)"\s*[|｜]?\s*>(.*?)</\s*[|｜]?\s*DSML\s*[|｜]?\s*invoke\s*[|｜]?\s*>`)
	// name is required; string="..." is optional (defaults to "true"); the
	// value is case-insensitive.
	dsmlRawParamRe = regexp.MustCompile(`(?is)<\s*[|｜]?\s*DSML\s*[|｜]?\s*parameter\s*[|｜]?\s*name="([^"]*)"\s*[|｜]?\s*(?:string="(true|false)"\s*[|｜]?\s*)?>(.*?)</\s*[|｜]?\s*DSML\s*[|｜]?\s*parameter\s*[|｜]?\s*>`)
	// Matches the end of a raw invoke (through its close tag), for buffer trimming.
	dsmlRawInvokeEndRe = regexp.MustCompile(`(?s).*?</\s*[|｜]?\s*DSML\s*[|｜]?\s*invoke\s*[|｜]?\s*>`)
)

// Canonical normalized form of the start tag; the hold-back check keeps text
// that is a PREFIX of it (a start tag split across chunks must not leak).
const dsmlStartTagNorm = "< DSML tool_calls>"

func isDSMLStartPrefix(norm string) bool {
	return strings.HasPrefix(dsmlStartTagNorm, norm)
}

// openAIStreamAdapter mirrors src/api-client/openai/streamAdapter.ts adaptOpenAIStreamToAnthropic.
type openAIStreamAdapter struct {
	model string

	started bool
	msgID   string

	currentContentIndex int
	toolBlocks          map[int]*openAIToolBlockState

	thinkingBlockOpen bool
	textBlockOpen     bool

	inputTokens  int
	outputTokens int
	cachedTokens int

	openBlockIndices map[int]struct{}

	// dsmlBuf accumulates content deltas that look like DSML tool calls
	// (DeepSeek V3.2/V4 grammar, e.g. <｜DSML｜tool_calls>...</｜DSML｜tool_calls>).
	// When a complete DSML block arrives it is parsed into tool_use events;
	// the text is NOT forwarded as a text_delta.
	dsmlBuf  strings.Builder
	dsmlOpen bool
}

type openAIToolBlockState struct {
	contentIndex int
	id           string
	name         string
	// Some OpenAI-compatible APIs send function.arguments as a JSON object/array instead of a string.
	// We marshal that once to a string and emit a single input_json_delta (OpenAI spec uses string fragments).
	emittedStructuredArgs bool
}

func newOpenAIStreamAdapter(model string) *openAIStreamAdapter {
	return &openAIStreamAdapter{
		model:               model,
		currentContentIndex: -1,
		toolBlocks:          make(map[int]*openAIToolBlockState),
		openBlockIndices:    make(map[int]struct{}),
		msgID:               openAIMessageStreamID(),
	}
}

func openAIMessageStreamID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return "msg_" + hex.EncodeToString(b[:])[:24]
}

func openAIToolPlaceholderID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return "toolu_" + hex.EncodeToString(b[:])[:24]
}

// openAIArgumentsFragment returns the next fragment to append to tool JSON input.
// OpenAI's schema uses a string for function.arguments; some proxies send a decoded object instead.
func openAIArgumentsFragment(fn map[string]any, st *openAIToolBlockState) (frag string, ok bool) {
	raw, exists := fn["arguments"]
	if !exists || raw == nil {
		return "", false
	}
	if s, okStr := raw.(string); okStr {
		return s, true
	}
	if st.emittedStructuredArgs {
		return "", false
	}
	st.emittedStructuredArgs = true
	b, err := json.Marshal(raw)
	if err != nil || len(b) == 0 || string(b) == "null" {
		return "", false
	}
	return string(b), true
}

func (a *openAIStreamAdapter) markOpen(idx int) {
	a.openBlockIndices[idx] = struct{}{}
}

func (a *openAIStreamAdapter) markClosed(idx int) {
	delete(a.openBlockIndices, idx)
}

func emitStreamObj(obj map[string]any, emit func(anthropicmessages.MessageStreamEvent) error) error {
	raw, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	var ev anthropicmessages.MessageStreamEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return err
	}
	return emit(ev)
}

func mapFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	case "content_filter":
		return "end_turn"
	default:
		return "end_turn"
	}
}

func (a *openAIStreamAdapter) applyUsageFromChunk(raw json.RawMessage) {
	if len(raw) == 0 || string(raw) == "null" {
		return
	}
	var u struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		PromptTokensDetails *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	}
	if err := json.Unmarshal(raw, &u); err != nil {
		return
	}
	if u.PromptTokens > 0 {
		a.inputTokens = u.PromptTokens
	}
	if u.CompletionTokens > 0 {
		a.outputTokens = u.CompletionTokens
	}
	if u.PromptTokensDetails != nil && u.PromptTokensDetails.CachedTokens > 0 {
		a.cachedTokens = u.PromptTokensDetails.CachedTokens
	}
}

func (a *openAIStreamAdapter) HandleChunk(chunkJSON []byte, emit func(anthropicmessages.MessageStreamEvent) error) error {
	var chunk struct {
		Choices []struct {
			Delta   json.RawMessage `json:"delta"`
			Message *struct {
				ReasoningContent *string `json:"reasoning_content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(chunkJSON, &chunk); err != nil {
		return fmt.Errorf("openai chunk: %w", err)
	}

	a.applyUsageFromChunk(chunk.Usage)

	if !a.started {
		a.started = true
		if err := emitStreamObj(map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id":            a.msgID,
				"type":          "message",
				"role":          "assistant",
				"content":       []any{},
				"model":         a.model,
				"stop_reason":   nil,
				"stop_sequence": nil,
				"usage": map[string]any{
					"input_tokens":                a.inputTokens,
					"output_tokens":               0,
					"cache_creation_input_tokens": 0,
					"cache_read_input_tokens":     a.cachedTokens,
				},
			},
		}, emit); err != nil {
			return err
		}
	}

	var delta struct {
		Content          *string         `json:"content"`
		ReasoningContent *string         `json:"reasoning_content"`
		ToolCalls        json.RawMessage `json:"tool_calls"`
	}
	if len(chunk.Choices) == 0 {
		return nil
	}
	ch0 := chunk.Choices[0]
	if len(ch0.Delta) > 0 && string(ch0.Delta) != "null" {
		_ = json.Unmarshal(ch0.Delta, &delta)
	}
	// Some gateways only attach chain-of-thought to choices[0].message (same as non-stream),
	// with an empty or omitted delta. Mirror src/services/api/openai/streamAdapter.ts.
	if (delta.ReasoningContent == nil || *delta.ReasoningContent == "") && ch0.Message != nil && ch0.Message.ReasoningContent != nil {
		c := *ch0.Message.ReasoningContent
		delta.ReasoningContent = &c
	}

	// reasoning_content → thinking
	// Empty string is a valid signal: DeepSeek v4 thinking mode sometimes
	// returns reasoning_content: "" when the model answers directly. The
	// empty thinking block must round-trip back to the API in subsequent
	// requests, otherwise DeepSeek rejects with 400.
	if delta.ReasoningContent != nil {
		if !a.thinkingBlockOpen {
			a.currentContentIndex++
			a.thinkingBlockOpen = true
			a.markOpen(a.currentContentIndex)
			if err := emitStreamObj(map[string]any{
				"type":  "content_block_start",
				"index": a.currentContentIndex,
				"content_block": map[string]any{
					"type": "thinking", "thinking": "", "signature": "",
				},
			}, emit); err != nil {
				return err
			}
		}
		if *delta.ReasoningContent != "" {
			if err := emitStreamObj(map[string]any{
				"type":  "content_block_delta",
				"index": a.currentContentIndex,
				"delta": map[string]any{
					"type": "thinking_delta", "thinking": *delta.ReasoningContent,
				},
			}, emit); err != nil {
				return err
			}
		}
	}

	if delta.Content != nil && *delta.Content != "" {
		content := *delta.Content
		// DSML tool-call interception: DeepSeek V3.2/V4 models may emit tool
		// calls as DSML text (e.g. <｜DSML｜tool_calls> / <|DSML|tool_calls>),
		// often mangled (tags/values spanning lines, glued attributes, dropped
		// bars). Detect on the NORMALIZED buffered text so a start tag that
		// straddles chunks is still caught.
		pending := a.dsmlBuf.String() + content
		normPending := normalizeDSML(pending)
		// Hold when already inside a block, when the buffer looks like the
		// start of a block (prefix of the canonical start tag, or a start tag
		// that just landed), or when a complete start tag appears ANYWHERE in
		// the pending text. The last case covers prose-then-block responses
		// (non-stream bodies): the whole chunk is buffered and the parser
		// extracts invokes; prose around the block is dropped once the wrapper
		// closes, which is the correct DSML semantics.
		hold := a.dsmlOpen || isDSMLStartPrefix(normPending) ||
			dsmlBlockStartRe.MatchString(normPending) ||
			dsmlRawStartRe.MatchString(pending)
		if hold {
			// Not yet in DSML mode but a complete start tag just appeared after
			// prose in this same chunk (non-stream bodies). Emit the prose as
			// text and buffer from the start tag onward. pending == content
			// here (empty buffer, not open), so prose before the tag is fresh
			// and must not be dropped.
			if !a.dsmlOpen && a.dsmlBuf.Len() == 0 {
				if loc := dsmlRawStartRe.FindStringIndex(pending); loc != nil && loc[0] > 0 {
					if lead := strings.TrimSpace(pending[:loc[0]]); lead != "" {
						if err := a.emitPlainText(lead, emit); err != nil {
							return err
						}
					}
					a.dsmlBuf.Reset()
					a.dsmlBuf.WriteString(pending[loc[0]:])
				} else {
					a.dsmlBuf.WriteString(content)
				}
			} else {
				a.dsmlBuf.WriteString(content)
			}
			a.dsmlOpen = true
			// Flush COMPLETE invokes incrementally — never wait for the wrapper
			// close tag (some models omit it). Each closed </DSML invoke>
			// becomes a tool_use immediately.
			for a.flushOneInvoke(emit) {
			}
			// Once the wrapper itself closed, clear the buffer so a trailing
			// </DSML tool_calls> (or prose after it) is not leaked as text.
			if dsmlRawEndRe.MatchString(a.dsmlBuf.String()) {
				a.dsmlBuf.Reset()
				a.dsmlOpen = false
			}
			return nil
		}
		// Normal text path (unchanged).
		if a.thinkingBlockOpen {
			if err := emitStreamObj(map[string]any{
				"type": "content_block_stop", "index": a.currentContentIndex,
			}, emit); err != nil {
				return err
			}
			a.markClosed(a.currentContentIndex)
			a.thinkingBlockOpen = false
		}
		if !a.textBlockOpen {
			a.currentContentIndex++
			a.textBlockOpen = true
			a.markOpen(a.currentContentIndex)
			if err := emitStreamObj(map[string]any{
				"type":  "content_block_start",
				"index": a.currentContentIndex,
				"content_block": map[string]any{
					"type": "text", "text": "",
				},
			}, emit); err != nil {
				return err
			}
		}
		if err := emitStreamObj(map[string]any{
			"type":  "content_block_delta",
			"index": a.currentContentIndex,
			"delta": map[string]any{
				"type": "text_delta", "text": content,
			},
		}, emit); err != nil {
			return err
		}
	}

	if len(delta.ToolCalls) > 0 && string(delta.ToolCalls) != "null" {
		var tcalls []map[string]any
		if err := json.Unmarshal(delta.ToolCalls, &tcalls); err == nil {
			for _, tc := range tcalls {
				tcIndex := 0
				if v, ok := tc["index"].(float64); ok {
					tcIndex = int(v)
				}
				if _, exists := a.toolBlocks[tcIndex]; !exists {
					if a.thinkingBlockOpen {
						if err := emitStreamObj(map[string]any{
							"type": "content_block_stop", "index": a.currentContentIndex,
						}, emit); err != nil {
							return err
						}
						a.markClosed(a.currentContentIndex)
						a.thinkingBlockOpen = false
					}
					if a.textBlockOpen {
						if err := emitStreamObj(map[string]any{
							"type": "content_block_stop", "index": a.currentContentIndex,
						}, emit); err != nil {
							return err
						}
						a.markClosed(a.currentContentIndex)
						a.textBlockOpen = false
					}
					a.currentContentIndex++
					toolID := openAIToolPlaceholderID()
					if idStr, ok := tc["id"].(string); ok && idStr != "" {
						toolID = idStr
					}
					toolName := ""
					if fn, ok := tc["function"].(map[string]any); ok {
						if n, ok := fn["name"].(string); ok {
							toolName = n
						}
					}
					a.toolBlocks[tcIndex] = &openAIToolBlockState{
						contentIndex: a.currentContentIndex,
						id:           toolID,
						name:         toolName,
					}
					a.markOpen(a.currentContentIndex)
					if err := emitStreamObj(map[string]any{
						"type":  "content_block_start",
						"index": a.currentContentIndex,
						"content_block": map[string]any{
							"type": "tool_use", "id": toolID, "name": toolName, "input": map[string]any{},
						},
					}, emit); err != nil {
						return err
					}
				}
				st := a.toolBlocks[tcIndex]
				if st == nil {
					continue
				}
				if fn, ok := tc["function"].(map[string]any); ok {
					if n, ok := fn["name"].(string); ok && n != "" {
						st.name = n
					}
					if arg, okFrag := openAIArgumentsFragment(fn, st); okFrag && arg != "" {
						if err := emitStreamObj(map[string]any{
							"type":  "content_block_delta",
							"index": st.contentIndex,
							"delta": map[string]any{
								"type": "input_json_delta", "partial_json": arg,
							},
						}, emit); err != nil {
							return err
						}
					}
				}
				if idStr, ok := tc["id"].(string); ok && idStr != "" {
					st.id = idStr
				}
			}
		}
	}

	if ch0.FinishReason != "" {
		if a.thinkingBlockOpen {
			if err := emitStreamObj(map[string]any{
				"type": "content_block_stop", "index": a.currentContentIndex,
			}, emit); err != nil {
				return err
			}
			a.markClosed(a.currentContentIndex)
			a.thinkingBlockOpen = false
		}
		if a.textBlockOpen {
			if err := emitStreamObj(map[string]any{
				"type": "content_block_stop", "index": a.currentContentIndex,
			}, emit); err != nil {
				return err
			}
			a.markClosed(a.currentContentIndex)
			a.textBlockOpen = false
		}
		idxSet := make(map[int]struct{})
		for _, st := range a.toolBlocks {
			if _, open := a.openBlockIndices[st.contentIndex]; open {
				idxSet[st.contentIndex] = struct{}{}
			}
		}
		toolIdxs := make([]int, 0, len(idxSet))
		for idx := range idxSet {
			toolIdxs = append(toolIdxs, idx)
		}
		sort.Ints(toolIdxs)
		for _, idx := range toolIdxs {
			if err := emitStreamObj(map[string]any{
				"type": "content_block_stop", "index": idx,
			}, emit); err != nil {
				return err
			}
			a.markClosed(idx)
		}
		stop := mapFinishReason(ch0.FinishReason)
		if err := emitStreamObj(map[string]any{
			"type": "message_delta",
			"delta": map[string]any{
				"stop_reason":   stop,
				"stop_sequence": nil,
				"usage": map[string]any{
					"input_tokens":  a.inputTokens,
					"output_tokens": a.outputTokens,
				},
			},
		}, emit); err != nil {
			return err
		}
		if err := emitStreamObj(map[string]any{"type": "message_stop"}, emit); err != nil {
			return err
		}
	}
	return nil
}

func (a *openAIStreamAdapter) FlushOpenBlocks(emit func(anthropicmessages.MessageStreamEvent) error) error {
	for idx := range a.openBlockIndices {
		if err := emitStreamObj(map[string]any{
			"type": "content_block_stop", "index": idx,
		}, emit); err != nil {
			return err
		}
	}
	a.openBlockIndices = make(map[int]struct{})

	// DSML tail safety: if a DSML block started but never closed (stream ended
	// mid-block), forward the buffered text as plain text instead of dropping it.
	if a.dsmlOpen && a.dsmlBuf.Len() > 0 {
		tail := a.dsmlBuf.String()
		a.dsmlBuf.Reset()
		a.dsmlOpen = false
		if err := a.emitPlainText(tail, emit); err != nil {
			return err
		}
	}
	return nil
}

// flushOneInvoke extracts the first complete </DSML invoke> from the DSML
// buffer and emits it as a tool_use. Returns true when one was flushed; the
// caller loops until false. The DSML wrapper close tag is NOT required — some
// models stop after the last invoke. Extraction uses RAW text (not the
// whitespace-collapsed normalization) so parameter values keep their pipes and
// other shell syntax; values are lightly cleaned (entity decode + newline
// folding).
func (a *openAIStreamAdapter) flushOneInvoke(emit func(anthropicmessages.MessageStreamEvent) error) bool {
	raw := a.dsmlBuf.String()
	m := dsmlRawInvokeRe.FindStringSubmatch(raw)
	if m == nil || len(m) < 3 {
		return false
	}
	toolName := strings.TrimSpace(m[1])
	paramsBody := m[2]
	args := map[string]any{}
	for _, pm := range dsmlRawParamRe.FindAllStringSubmatch(paramsBody, -1) {
		if len(pm) < 4 {
			continue
		}
		name := strings.TrimSpace(pm[1])
		// string attr optional: absent, "true", or "TRUE" → keep string.
		// Only "false"/"FALSE" coerces the value away from string.
		isString := strings.ToLower(pm[2]) != "false"
		val := cleanDSMLValue(pm[3])
		if isString {
			args[name] = val
		} else {
			// Coerce loosely: try JSON (numbers/bools/objects/arrays); fall back to string.
			var jv any
			if err := json.Unmarshal([]byte(val), &jv); err == nil {
				args[name] = jv
			} else {
				args[name] = val
			}
		}
	}
	if err := a.emitOneInvoke(toolName, args, emit); err != nil {
		return false
	}
	// Remove the flushed invoke from the buffer: drop everything up to and
	// including the first raw </DSML invoke> close tag.
	end := dsmlRawInvokeEndRe.FindStringIndex(raw)
	if end == nil {
		a.dsmlBuf.Reset()
		return true
	}
	a.dsmlBuf.Reset()
	a.dsmlBuf.WriteString(raw[end[1]:])
	return true
}

// cleanDSMLValue decodes XML entities and folds internal newlines to single
// spaces (a value spanning lines still round-trips; shell treats newlines as
// spaces). Internal pipes are untouched.
func cleanDSMLValue(v string) string {
	v = strings.ReplaceAll(v, "&amp;", "&")
	v = strings.ReplaceAll(v, "&lt;", "<")
	v = strings.ReplaceAll(v, "&gt;", ">")
	v = strings.ReplaceAll(v, "&quot;", `"`)
	v = strings.ReplaceAll(v, "&apos;", "'")
	return strings.TrimSpace(v)
}

// emitOneInvoke emits a single tool_use event sequence
// (content_block_start → input_json_delta → content_block_stop).
func (a *openAIStreamAdapter) emitOneInvoke(toolName string, args map[string]any, emit func(anthropicmessages.MessageStreamEvent) error) error {
	// Close any open thinking/text block before emitting tool_use.
	if a.thinkingBlockOpen {
		if err := emitStreamObj(map[string]any{
			"type": "content_block_stop", "index": a.currentContentIndex,
		}, emit); err != nil {
			return err
		}
		a.markClosed(a.currentContentIndex)
		a.thinkingBlockOpen = false
	}
	if a.textBlockOpen {
		if err := emitStreamObj(map[string]any{
			"type": "content_block_stop", "index": a.currentContentIndex,
		}, emit); err != nil {
			return err
		}
		a.markClosed(a.currentContentIndex)
		a.textBlockOpen = false
	}

	a.currentContentIndex++
	toolID := openAIToolPlaceholderID()
	a.toolBlocks[a.currentContentIndex] = &openAIToolBlockState{
		contentIndex: a.currentContentIndex,
		id:           toolID,
		name:         toolName,
	}
	a.markOpen(a.currentContentIndex)
	if err := emitStreamObj(map[string]any{
		"type":  "content_block_start",
		"index": a.currentContentIndex,
		"content_block": map[string]any{
			"type": "tool_use", "id": toolID, "name": toolName, "input": map[string]any{},
		},
	}, emit); err != nil {
		return err
	}
	argsJSON, _ := json.Marshal(args)
	if len(argsJSON) > 0 && string(argsJSON) != "{}" {
		if err := emitStreamObj(map[string]any{
			"type":  "content_block_delta",
			"index": a.currentContentIndex,
			"delta": map[string]any{
				"type": "input_json_delta", "partial_json": string(argsJSON),
			},
		}, emit); err != nil {
			return err
		}
	}
	if err := emitStreamObj(map[string]any{
		"type": "content_block_stop", "index": a.currentContentIndex,
	}, emit); err != nil {
		return err
	}
	a.markClosed(a.currentContentIndex)
	return nil
}

// emitPlainText forwards a plain-text delta, opening a text block if needed.
func (a *openAIStreamAdapter) emitPlainText(text string, emit func(anthropicmessages.MessageStreamEvent) error) error {
	if text == "" {
		return nil
	}
	if a.thinkingBlockOpen {
		if err := emitStreamObj(map[string]any{
			"type": "content_block_stop", "index": a.currentContentIndex,
		}, emit); err != nil {
			return err
		}
		a.markClosed(a.currentContentIndex)
		a.thinkingBlockOpen = false
	}
	if !a.textBlockOpen {
		a.currentContentIndex++
		a.textBlockOpen = true
		a.markOpen(a.currentContentIndex)
		if err := emitStreamObj(map[string]any{
			"type":  "content_block_start",
			"index": a.currentContentIndex,
			"content_block": map[string]any{
				"type": "text", "text": "",
			},
		}, emit); err != nil {
			return err
		}
	}
	return emitStreamObj(map[string]any{
		"type":  "content_block_delta",
		"index": a.currentContentIndex,
		"delta": map[string]any{
			"type": "text_delta", "text": text,
		},
	}, emit)
}

func ReplayOpenAIStreamChatResponse(sseBody []byte, model string, emit func(anthropicmessages.MessageStreamEvent) error) error {
	ad := newOpenAIStreamAdapter(model)
	if err := anthropicmessages.ReadSSE(bytes.NewReader(sseBody), func(data []byte) error {
		if len(data) == 0 || string(data) == "[DONE]" {
			return nil
		}
		return ad.HandleChunk(data, emit)
	}); err != nil {
		return err
	}
	return ad.FlushOpenBlocks(emit)
}

// NormalizeOpenAINonStreamChatBodyToolCallsLoose rewrites choices[0].message.tool_calls so sloppy
// OpenAI-compatible models still replay through [ReplayOpenAINonStreamChatResponse]:
// wraps a lone tool-call object as an array, lifts top-level name/arguments into function.{name,arguments},
// coerces non-string arguments to a JSON string, fills missing id/type/index.
func NormalizeOpenAINonStreamChatBodyToolCallsLoose(respBody []byte) []byte {
	var root map[string]any
	if err := json.Unmarshal(respBody, &root); err != nil {
		return respBody
	}
	choices, ok := root["choices"].([]any)
	if !ok || len(choices) == 0 {
		return respBody
	}
	ch0, ok := choices[0].(map[string]any)
	if !ok {
		return respBody
	}
	msg, ok := ch0["message"].(map[string]any)
	if !ok {
		return respBody
	}
	tcRaw, ok := msg["tool_calls"]
	if !ok || tcRaw == nil {
		return respBody
	}
	normalized := normalizeOpenAIToolCallsValue(tcRaw)
	if normalized == nil {
		return respBody
	}
	msg["tool_calls"] = normalized
	out, err := json.Marshal(root)
	if err != nil {
		return respBody
	}
	return out
}

func normalizeOpenAIToolCallsValue(v any) any {
	if m, ok := v.(map[string]any); ok {
		if _, fnOK := m["function"].(map[string]any); fnOK {
			return []any{m}
		}
		if _, nameOK := m["name"].(string); nameOK {
			return []any{m}
		}
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(arr))
	for i, item := range arr {
		tc, ok := item.(map[string]any)
		if !ok {
			continue
		}
		normalizeOpenAIToolCallMapInPlace(tc, i)
		out = append(out, tc)
	}
	return out
}

func normalizeOpenAIToolCallMapInPlace(tc map[string]any, order int) {
	if fn, ok := tc["function"].(map[string]any); ok {
		if arg, has := fn["arguments"]; has {
			fn["arguments"] = openAINonStreamCoerceArgsString(arg)
		} else if arg, has := fn["input"]; has {
			fn["arguments"] = openAINonStreamCoerceArgsString(arg)
			delete(fn, "input")
		} else {
			fn["arguments"] = "{}"
		}
		if n, ok := fn["name"].(string); !ok || strings.TrimSpace(n) == "" {
			if n2, ok := tc["name"].(string); ok && strings.TrimSpace(n2) != "" {
				fn["name"] = n2
				delete(tc, "name")
			}
		}
	} else {
		fn := map[string]any{}
		if n, ok := tc["name"].(string); ok {
			fn["name"] = n
		}
		arg := tc["arguments"]
		if arg == nil {
			arg = tc["args"]
		}
		if arg == nil {
			arg = tc["input"]
		}
		fn["arguments"] = openAINonStreamCoerceArgsString(arg)
		tc["function"] = fn
		delete(tc, "name")
		delete(tc, "arguments")
		delete(tc, "args")
		delete(tc, "input")
	}
	if id, ok := tc["id"].(string); !ok || strings.TrimSpace(id) == "" {
		tc["id"] = openAIToolPlaceholderID()
	}
	if typ, ok := tc["type"].(string); !ok || strings.TrimSpace(typ) == "" {
		tc["type"] = "function"
	}
	if _, ok := tc["index"].(float64); !ok {
		tc["index"] = float64(order)
	}
}

func openAINonStreamCoerceArgsString(v any) string {
	if v == nil {
		return "{}"
	}
	if s, ok := v.(string); ok {
		if strings.TrimSpace(s) == "" {
			return "{}"
		}
		return s
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 || string(b) == "null" {
		return "{}"
	}
	return string(b)
}

// ReplayOpenAINonStreamChatResponse converts a full POST /v1/chat/completions JSON body (stream:false)
// into the same Anthropic-shaped stream events [openAIStreamAdapter] would emit, so
// [assistantStreamAccumulator] and the streaming tool executor stay unchanged.
func ReplayOpenAINonStreamChatResponse(respBody []byte, model string, emit func(anthropicmessages.MessageStreamEvent) error) error {
	var head struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Choices []json.RawMessage `json:"choices"`
		Usage   json.RawMessage   `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &head); err != nil {
		return fmt.Errorf("openai non-stream json: %w", err)
	}
	if head.Error != nil && strings.TrimSpace(head.Error.Message) != "" {
		return fmt.Errorf("openai non-stream api error: %s", head.Error.Message)
	}
	if len(head.Choices) == 0 {
		return fmt.Errorf("openai non-stream: empty choices")
	}
	var ch struct {
		Message struct {
			Content          json.RawMessage  `json:"content"`
			ReasoningContent *string          `json:"reasoning_content"`
			ToolCalls        []map[string]any `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}
	if err := json.Unmarshal(head.Choices[0], &ch); err != nil {
		return fmt.Errorf("openai non-stream choice: %w", err)
	}

	ad := newOpenAIStreamAdapter(model)
	kick, err := json.Marshal(map[string]any{
		"choices": []map[string]any{{"index": 0, "delta": map[string]any{}}},
		"usage":   head.Usage,
	})
	if err != nil {
		return err
	}
	if err := ad.HandleChunk(kick, emit); err != nil {
		return err
	}

	// DeepSeek reasoner (and similar): non-stream body includes message.reasoning_content (chain-of-thought).
	// Mirror streaming deltas: emit reasoning before visible content (see HandleChunk reasoning_content branch).
	if ch.Message.ReasoningContent != nil && strings.TrimSpace(*ch.Message.ReasoningContent) != "" {
		rc, err := json.Marshal(map[string]any{
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{"reasoning_content": *ch.Message.ReasoningContent},
			}},
		})
		if err != nil {
			return err
		}
		if err := ad.HandleChunk(rc, emit); err != nil {
			return err
		}
	}

	var textPieces []string
	if len(ch.Message.Content) > 0 && string(ch.Message.Content) != "null" {
		var s string
		if err := json.Unmarshal(ch.Message.Content, &s); err == nil {
			if strings.TrimSpace(s) != "" {
				textPieces = append(textPieces, s)
			}
		} else {
			var parts []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(ch.Message.Content, &parts); err == nil {
				for _, p := range parts {
					if p.Type == "text" && p.Text != "" {
						textPieces = append(textPieces, p.Text)
					}
				}
			}
		}
	}
	for _, piece := range textPieces {
		b, err := json.Marshal(map[string]any{
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{"content": piece},
			}},
		})
		if err != nil {
			return err
		}
		if err := ad.HandleChunk(b, emit); err != nil {
			return err
		}
	}

	if len(ch.Message.ToolCalls) > 0 {
		tcalls := make([]map[string]any, 0, len(ch.Message.ToolCalls))
		for i, tc := range ch.Message.ToolCalls {
			if tc == nil {
				continue
			}
			if _, ok := tc["index"]; !ok {
				tc["index"] = float64(i)
			}
			tcalls = append(tcalls, tc)
		}
		if len(tcalls) > 0 {
			b, err := json.Marshal(map[string]any{
				"choices": []map[string]any{{
					"index": 0,
					"delta": map[string]any{"tool_calls": tcalls},
				}},
			})
			if err != nil {
				return err
			}
			if err := ad.HandleChunk(b, emit); err != nil {
				return err
			}
		}
	}

	finish := strings.TrimSpace(ch.FinishReason)
	if finish == "" {
		if len(ch.Message.ToolCalls) > 0 {
			finish = "tool_calls"
		} else {
			finish = "stop"
		}
	}
	endObj := map[string]any{
		"choices": []map[string]any{{
			"index":         0,
			"delta":         map[string]any{},
			"finish_reason": finish,
		}},
	}
	if len(head.Usage) > 0 && string(head.Usage) != "null" {
		endObj["usage"] = json.RawMessage(head.Usage)
	}
	endB, err := json.Marshal(endObj)
	if err != nil {
		return err
	}
	if err := ad.HandleChunk(endB, emit); err != nil {
		return err
	}
	return ad.FlushOpenBlocks(emit)
}
