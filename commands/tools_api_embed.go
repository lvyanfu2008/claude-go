package commands

import _ "embed"

// ToolsAPIJSON is data/tools_api.json from scripts/export-tools-registry-json.ts (see meta.source).
//
//go:embed data/tools_api.json
var ToolsAPIJSON []byte
