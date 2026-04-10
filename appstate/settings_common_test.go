package appstate

import (
	"encoding/json"
	"testing"
)

func TestParseSettingsCommon(t *testing.T) {
	raw := json.RawMessage(`{"model":"claude-sonnet-4","language":"en","extra":true}`)
	s, err := ParseSettingsCommon(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.Model == nil || *s.Model != "claude-sonnet-4" {
		t.Fatalf("%v", s.Model)
	}
	if s.Language == nil || *s.Language != "en" {
		t.Fatalf("%v", s.Language)
	}
}
