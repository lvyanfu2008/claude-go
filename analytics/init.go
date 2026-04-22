package analytics

import "goc/diagnostics"

// InitDefaultPipeline runs [goc/diagnostics.InitAnalytics] then [InitializeAnalyticsSink].
// Use this once at process startup when you want TS-style logEvent → writers behavior.
func InitDefaultPipeline() {
	diagnostics.InitAnalytics()
	InitializeAnalyticsSink()
}
