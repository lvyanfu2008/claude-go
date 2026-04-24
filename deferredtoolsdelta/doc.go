// Package deferredtoolsdelta holds shared logic for TypeScript parity around deferred-tool
// discovery UX: when deferred tool names are announced via <system-reminder> / delta attachments
// vs <available-deferred-tools> prepends, and the ToolSearch tool description shown to the model
// (src/tools/ToolSearchTool/prompt.ts getPrompt / getToolLocationHint; src/utils/toolSearch.ts isDeferredToolsDeltaEnabled).
//
// It lives at module root (not under goc/internal/toolsearch) to avoid an import cycle:
// goc/tools/toolpool → toolsearch → anthropic → goc/tools/toolpool.
package deferredtoolsdelta
