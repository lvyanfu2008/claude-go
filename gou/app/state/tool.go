package state

import "goc/tools/toolresultpersist"

type Tool struct {
	ResultState         *toolresultpersist.ContentReplacementState
	MCPCommandsJSONPath string
	MCPToolsJSONPath    string
}
