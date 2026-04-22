// Package memoize is a port of claude-code src/utils/memoize.ts: TTL stale-while-revalidate,
// async TTL with cold-miss coalescing, and LRU by string key. Use in any goc subpackage
// (mirrors the shared TS util).
package memoize
