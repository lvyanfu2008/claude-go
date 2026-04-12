package anthropic

import (
	"strings"

	"goc/ccb-engine/diaglog"
)

// LogToolsLoaded appends one line to the diagnostic log (see [diaglog.Line]): tool count and names.
// contextTag identifies the load site (e.g. "gou-demo", "socketserve", "ccb-engine-cli").
// requestID may be empty when not applicable.
func LogToolsLoaded(contextTag, requestID string, source string, tools []ToolDefinition) {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		n := strings.TrimSpace(t.Name)
		if n == "" {
			n = "(unnamed)"
		}
		names = append(names, n)
	}
	rid := requestID
	if rid == "" {
		rid = "-"
	}
	diaglog.Line("[ccb-engine tools] %s request_id=%q source=%s count=%d names=[%s]",
		contextTag, rid, source, len(tools), strings.Join(names, ", "))
}
