package autodream

import (
	"fmt"
	"strings"
)

const (
	entrypointName    = "MEMORY.md"
	maxEntrypointLines = 200
	dirExistsGuidance = "This directory already exists — write to it directly with the Write tool (do not run mkdir or check for its existence)."
)

// BuildConsolidationPrompt mirrors buildConsolidationPrompt() in
// src/services/autoDream/consolidationPrompt.ts.
func BuildConsolidationPrompt(memoryRoot, transcriptDir, extra string) string {
	var b strings.Builder

	b.WriteString("# Dream: Memory Consolidation\n\n")
	b.WriteString("You are performing a dream — a reflective pass over your memory files. Synthesize what you've learned recently into durable, well-organized memories so that future sessions can orient quickly.\n\n")
	b.WriteString(fmt.Sprintf("Memory directory: `%s`\n", memoryRoot))
	b.WriteString(dirExistsGuidance + "\n\n")
	b.WriteString(fmt.Sprintf("Session transcripts: `%s` (large JSONL files — grep narrowly, don't read whole files)\n", transcriptDir))
	b.WriteString("\n---\n\n")
	b.WriteString("## Phase 1 — Orient\n\n")
	b.WriteString("- `ls` the memory directory to see what already exists\n")
	b.WriteString(fmt.Sprintf("- Read `%s` to understand the current index\n", entrypointName))
	b.WriteString("- Skim existing topic files so you improve them rather than creating duplicates\n")
	b.WriteString("- If `logs/` or `sessions/` subdirectories exist (assistant-mode layout), review recent entries there\n\n")
	b.WriteString("## Phase 2 — Gather recent signal\n\n")
	b.WriteString("Look for new information worth persisting. Sources in rough priority order:\n\n")
	b.WriteString("1. **Daily logs** (`logs/YYYY/MM/YYYY-MM-DD.md`) if present — these are the append-only stream\n")
	b.WriteString("2. **Existing memories that drifted** — facts that contradict something you see in the codebase now\n")
	b.WriteString("3. **Transcript search** — if you need specific context (e.g., \"what was the error message from yesterday's build failure?\"), grep the JSONL transcripts for narrow terms:\n")
	b.WriteString(fmt.Sprintf("   `grep -rn \"<narrow term>\" %s/ --include=\"*.jsonl\" | tail -50`\n\n", transcriptDir))
	b.WriteString("Don't exhaustively read transcripts. Look only for things you already suspect matter.\n\n")
	b.WriteString("## Phase 3 — Consolidate\n\n")
	b.WriteString("For each thing worth remembering, write or update a memory file at the top level of the memory directory. Use the memory file format and type conventions from your system prompt's auto-memory section — it's the source of truth for what to save, how to structure it, and what NOT to save.\n\n")
	b.WriteString("Focus on:\n")
	b.WriteString("- Merging new signal into existing topic files rather than creating near-duplicates\n")
	b.WriteString("- Converting relative dates (\"yesterday\", \"last week\") to absolute dates so they remain interpretable after time passes\n")
	b.WriteString("- Deleting contradicted facts — if today's investigation disproves an old memory, fix it at the source\n\n")
	b.WriteString("## Phase 4 — Prune and index\n\n")
	b.WriteString(fmt.Sprintf("Update `%s` so it stays under %d lines AND under ~25KB. It's an **index**, not a dump — each entry should be one line under ~150 characters: `- [Title](file.md) — one-line hook`. Never write memory content directly into it.\n\n", entrypointName, maxEntrypointLines))
	b.WriteString("- Remove pointers to memories that are now stale, wrong, or superseded\n")
	b.WriteString("- Demote verbose entries: if an index line is over ~200 chars, it's carrying content that belongs in the topic file — shorten the line, move the detail\n")
	b.WriteString("- Add pointers to newly important memories\n")
	b.WriteString("- Resolve contradictions — if two files disagree, fix the wrong one\n\n")
	b.WriteString("---\n\n")
	b.WriteString("Return a brief summary of what you consolidated, updated, or pruned. If nothing changed (memories are already tight), say so. If no sessions are listed above (this is a manual /dream), review the memory directory as-is.\n")
	if extra != "" {
		b.WriteString(fmt.Sprintf("\n## Additional context\n\n%s\n", extra))
	}

	return strings.TrimSpace(b.String())
}
