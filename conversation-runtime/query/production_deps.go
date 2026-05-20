package query

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"goc/compactservice"
)

// ProductionDeps mirrors src/conversation-runtime/queryPipeline/deps.ts productionDeps().
// CallModel / Microcompact are nil by default; hosts may set them (see [preflight.go] types).
//
// Autocompact is wired to the default compactservice adapter — TS parity: productionDeps()
// passes autoCompactIfNeeded directly. The adapter uses a direct no-tools streaming call
// for the summary, and defaults to no-op for pre/post-compact hooks and attachment regeneration
// until those subsystems land Go parity (see compactservice/doc.go for the full list).
//
// trySMCompact is an optional [compactservice.TrySessionMemoryCompactFn] that enables
// session-memory compaction. Hosts that have a sessionmemory.State should provide a closure
// that bridges to sessionmemory.TrySessionMemoryCompaction. Pass nil to disable (harmless —
// API-based compaction is used as fallback).
func ProductionDeps(trySMCompact compactservice.TrySessionMemoryCompactFn, onCompactPhase func(string)) QueryDeps {
	return QueryDeps{
		NewUUID:     randomUUID,
		Autocompact: NewCompactAdapter(trySMCompact, onCompactPhase),
	}
}

// RandomUUID generates a random UUID v4 string (exported reference for packages that
// need a [QueryDeps.NewUUID]-compatible fallback).
func RandomUUID() string {
	return randomUUID()
}

func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%s",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		hex.EncodeToString(b[10:16]),
	)
}
