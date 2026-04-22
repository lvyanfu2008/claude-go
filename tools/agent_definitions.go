package tools

import (
	"strings"

	"goc/agents/builtin"
)

func LoadAgentDefinitions() []AgentDefinition {
	return LoadAgentDefinitionsForCwd(strings.TrimSpace(getenv("PWD")))
}

func LoadAgentDefinitionsForCwd(cwd string) []AgentDefinition {
	return LoadAgentDefinitionsReport(strings.TrimSpace(cwd)).ActiveAgents
}

func builtinConfigFromEnv() builtin.Config { return builtin.ConfigFromEnv() }
func getBuiltinAgents(cfg builtin.Config) []builtin.BuiltinAgent {
	return builtin.GetBuiltInAgents(cfg, builtin.GuideContext{})
}

func ResolveAgentDefinition(all []AgentDefinition, subagentType string) AgentDefinition {
	want := strings.TrimSpace(subagentType)
	if want == "" {
		want = "general-purpose"
	}
	for _, a := range all {
		if strings.EqualFold(a.AgentType, want) {
			return a
		}
	}
	for _, a := range all {
		if strings.EqualFold(a.AgentType, "general-purpose") {
			return a
		}
	}
	return AgentDefinition{
		AgentType: "general-purpose",
		WhenToUse: "General-purpose fallback agent.",
		Tools:     []string{"*"},
		Source:    "built-in",
	}
}

func FilterAgentsByRequiredMCPServers(all []AgentDefinition, availableServers []string) []AgentDefinition {
	if len(availableServers) == 0 {
		availableServers = []string{}
	}
	var out []AgentDefinition
	for _, a := range all {
		if len(a.RequiredMcpServers) == 0 {
			out = append(out, a)
			continue
		}
		ok := true
		for _, want := range a.RequiredMcpServers {
			w := strings.ToLower(strings.TrimSpace(want))
			if w == "" {
				continue
			}
			matched := false
			for _, got := range availableServers {
				if strings.Contains(strings.ToLower(got), w) {
					matched = true
					break
				}
			}
			if !matched {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, a)
		}
	}
	return out
}

func AgentMeetsRequiredMCPServers(a AgentDefinition, availableServers []string) bool {
	if len(a.RequiredMcpServers) == 0 {
		return true
	}
	if len(availableServers) == 0 {
		return false
	}
	for _, want := range a.RequiredMcpServers {
		w := strings.ToLower(strings.TrimSpace(want))
		if w == "" {
			continue
		}
		matched := false
		for _, got := range availableServers {
			if strings.Contains(strings.ToLower(got), w) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func ResolveAllowedTools(a AgentDefinition, available []string) []string {
	if len(a.Tools) == 0 && len(a.DisallowedTools) == 0 {
		return append([]string(nil), available...)
	}
	deny := map[string]struct{}{}
	for _, t := range a.DisallowedTools {
		deny[t] = struct{}{}
	}
	if len(a.Tools) > 0 {
		var out []string
		for _, t := range a.Tools {
			if _, blocked := deny[t]; !blocked {
				out = append(out, t)
			}
		}
		return out
	}
	var out []string
	for _, t := range available {
		if _, blocked := deny[t]; !blocked {
			out = append(out, t)
		}
	}
	return out
}
