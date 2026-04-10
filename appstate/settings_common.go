package appstate

import "encoding/json"

// SettingsCommon holds a small, frequently-read subset of merged settings (TS SettingsJson).
// Use [ParseSettingsCommon] on [AppState].Settings; unknown keys are ignored.
type SettingsCommon struct {
	Model                 *string `json:"model,omitempty"`
	Language              *string `json:"language,omitempty"`
	OutputStyle           *string `json:"outputStyle,omitempty"`
	AlwaysThinkingEnabled *bool   `json:"alwaysThinkingEnabled,omitempty"`
}

// ParseSettingsCommon decodes known fields from merged settings JSON (empty / null → zero struct).
func ParseSettingsCommon(raw json.RawMessage) (SettingsCommon, error) {
	var s SettingsCommon
	if len(raw) == 0 || string(raw) == "null" {
		return s, nil
	}
	err := json.Unmarshal(raw, &s)
	return s, err
}
