package analytics

import "strings"

// StripProtoFields removes keys prefixed with "_PROTO_" from metadata before
// general-access sinks (TS: services/analytics/index.ts stripProtoFields).
func StripProtoFields(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return metadata
	}
	hasProto := false
	for k := range metadata {
		if strings.HasPrefix(k, "_PROTO_") {
			hasProto = true
			break
		}
	}
	if !hasProto {
		return metadata
	}
	out := make(map[string]any, len(metadata))
	for k, v := range metadata {
		if strings.HasPrefix(k, "_PROTO_") {
			continue
		}
		out[k] = v
	}
	return out
}
