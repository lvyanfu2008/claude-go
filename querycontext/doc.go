// Package querycontext mirrors src/utils/queryContext.ts fetchSystemPromptParts and its
// dependencies (getUserContext, getSystemContext) for Go-only runtimes (gou-demo, ccb-engine).
//
// User/system context and git status use session-level memoization matching lodash memoize
// in src/context.ts. Use [ClearAllContextCaches], [ClearUserAndSystemContextCaches], or
// [ClearUserContextCache] to invalidate (TS clearCaches / setSystemPromptInjection / compact).
//
// Full parity with TS claudemd.ts / getSystemPrompt is not attempted; see package comments on each builder.
package querycontext
