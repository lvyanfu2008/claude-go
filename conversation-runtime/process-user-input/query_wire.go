package processuserinput

import (
	"goc/conversation-runtime/query"
	"goc/toolexecution"
)

// ApplyQueryHostEnvGates sets [query.QueryParams.StreamingParity] when [query.BuildQueryConfig] env gates
// request the HTTP SSE parity path (GOU_QUERY_STREAMING_PARITY or GOU_DEMO_STREAMING_TOOL_EXECUTION).
// Call from hosts (for example gou-demo) before [query.Query] when merged settings env should drive parity.
func ApplyQueryHostEnvGates(qp *query.QueryParams) {
	if qp == nil {
		return
	}
	cfg := query.BuildQueryConfig()
	if query.StreamingParityPathEnabled(cfg) {
		qp.StreamingParity = true
	}
}

// WireToolexecutionFromProcessUserInput copies [ProcessUserInputParams.CanUseTool] onto
// qp.Deps.ToolexecutionDeps.QueryCanUseTool when still nil, allocating [query.ProductionDeps] if qp.Deps is nil.
func WireToolexecutionFromProcessUserInput(qp *query.QueryParams, p *ProcessUserInputParams) {
	if qp == nil || p == nil || p.CanUseTool == nil {
		return
	}
	if qp.Deps == nil {
		d := query.ProductionDeps()
		qp.Deps = &d
	}
	if qp.Deps.ToolexecutionDeps.QueryCanUseTool == nil {
		qp.Deps.ToolexecutionDeps.QueryCanUseTool = toolexecution.LegacyBoolQueryGate(p.CanUseTool)
	}
}
