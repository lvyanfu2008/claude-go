// Package querycontext mirrors src/utils/queryContext.ts fetchSystemPromptParts and its
// dependencies (getUserContext, getSystemContext) for Go-only runtimes (gou-demo, ccb-engine).
//
// User/system context and git status use session-level memoization matching lodash memoize
// in src/context.ts. Use [ClearAllContextCaches], [ClearUserAndSystemContextCaches], or
// [ClearUserContextCache] to invalidate. [ClearAllContextCaches] also calls
// [claudemd.ResetMemoryFilesCache](session_start) (getMemoryFiles / clear session caches). For
// getMemoryFiles only, use [claudemd.ClearMemoryFileCaches] or [claudemd.ResetMemoryFilesCache].
//
// Full parity with TS claudemd.ts / getSystemPrompt is not attempted; see package comments on each builder.
package querycontext
