// Package anthropicmessages implements Anthropic Messages API SSE streaming for callers
// outside goc/internal/anthropic (e.g. conversation-runtime/query).
package anthropicmessages

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ErrMessageStreamDone is returned from the SSE reader when type message_stop is processed.
var ErrMessageStreamDone = errors.New("anthropicmessages: message_stop received")

// MessageStreamEvent is one decoded `data:` JSON object from Messages SSE.
type MessageStreamEvent struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"-"`
}

// UnmarshalJSON captures the full object and sets Type for dispatch.
func (e *MessageStreamEvent) UnmarshalJSON(b []byte) error {
	e.Raw = append(json.RawMessage(nil), b...)
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return err
	}
	e.Type = head.Type
	return nil
}

// ReadSSE invokes yield for each complete SSE event payload (concatenated data: lines).
func ReadSSE(r io.Reader, yield func(data []byte) error) error {
	br := bufio.NewReader(r)
	var dataParts [][]byte
	flush := func() error {
		if len(dataParts) == 0 {
			return nil
		}
		payload := bytes.Join(dataParts, []byte("\n"))
		dataParts = dataParts[:0]
		if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
			return nil
		}
		if err := yield(payload); err != nil {
			if errors.Is(err, ErrMessageStreamDone) {
				return ErrMessageStreamDone
			}
			return err
		}
		return nil
	}

	for {
		line, err := br.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				if err := flush(); err != nil {
					if errors.Is(err, ErrMessageStreamDone) {
						return nil
					}
					return err
				}
				return nil
			}
			return err
		}
		line = bytes.TrimSuffix(line, []byte("\n"))
		line = bytes.TrimSuffix(line, []byte("\r"))
		if len(line) == 0 {
			if err := flush(); err != nil {
				if errors.Is(err, ErrMessageStreamDone) {
					return nil
				}
				return err
			}
			continue
		}
		if line[0] == ':' {
			continue
		}
		if bytes.HasPrefix(line, []byte("data:")) {
			dataParts = append(dataParts, bytes.TrimSpace(line[5:]))
			continue
		}
	}
}

// DecodeStreamPayload unmarshals one SSE data line into [MessageStreamEvent].
func DecodeStreamPayload(data []byte) (MessageStreamEvent, error) {
	var ev MessageStreamEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return ev, err
	}
	return ev, nil
}

// MarshalJSONNoEscapeHTML matches JSON.stringify for request bodies.
func MarshalJSONNoEscapeHTML(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// ProcessStreamPayloads parses JSON and handles message_stop / error types for PostStream.
func ProcessStreamPayloads(data []byte, emit func(MessageStreamEvent) error) error {
	ev, err := DecodeStreamPayload(data)
	if err != nil {
		return fmt.Errorf("sse json: %w", err)
	}
	if ev.Type == "error" {
		return fmt.Errorf("stream error: %s", string(ev.Raw))
	}
	if err := emit(ev); err != nil {
		return err
	}
	if ev.Type == "message_stop" {
		return ErrMessageStreamDone
	}
	return nil
}
