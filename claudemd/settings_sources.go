package claudemd

import (
	"os"
	"strings"
)

// SettingSource mirrors TS SettingSource strings used by claudemd.
const (
	SourceUserSettings    = "userSettings"
	SourceProjectSettings = "projectSettings"
	SourceLocalSettings   = "localSettings"
	SourceFlagSettings    = "flagSettings"
	SourcePolicySettings  = "policySettings"
)

// defaultAllowed mirrors bootstrap STATE.allowedSettingSources initial value.
// settings.go.json is Go-only and is not a TS setting source.
var defaultAllowed = []string{
	SourceUserSettings,
	SourceProjectSettings,
	SourceLocalSettings,
	SourceFlagSettings,
	SourcePolicySettings,
}

// AllowedSettingSources returns getAllowedSettingSources equivalent.
// CLAUDE_CODE_SETTING_SOURCES: comma list user,project,local
// Value "isolated" → empty allow-list (SDK-style: only policy+flag still enabled via EnabledSettingSources).
func AllowedSettingSources() []string {
	raw := strings.TrimSpace(os.Getenv("CLAUDE_CODE_SETTING_SOURCES"))
	if raw == "" {
		out := make([]string, len(defaultAllowed))
		copy(out, defaultAllowed)
		return out
	}
	if strings.EqualFold(raw, "isolated") {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		switch strings.TrimSpace(strings.ToLower(part)) {
		case "user":
			out = append(out, SourceUserSettings)
		case "project":
			out = append(out, SourceProjectSettings)
		case "local":
			out = append(out, SourceLocalSettings)
		case "flag":
			out = append(out, SourceFlagSettings)
		case "policy":
			out = append(out, SourcePolicySettings)
		}
	}
	if len(out) == 0 {
		return defaultAllowed
	}
	return dedupeKeepOrder(out)
}

// EnabledSettingSources matches getEnabledSettingSources (allowed ∪ policy ∪ flag, stable order).
func EnabledSettingSources() []string {
	allowed := AllowedSettingSources()
	seen := map[string]struct{}{}
	var order []string
	add := func(s string) {
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		order = append(order, s)
	}
	for _, s := range allowed {
		add(s)
	}
	add(SourcePolicySettings)
	add(SourceFlagSettings)
	return order
}

// IsSettingSourceEnabled mirrors isSettingSourceEnabled.
func IsSettingSourceEnabled(source string) bool {
	for _, s := range EnabledSettingSources() {
		if s == source {
			return true
		}
	}
	return false
}

func dedupeKeepOrder(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
