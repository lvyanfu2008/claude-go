package slashresolve

import (
	"fmt"
	"path/filepath"
	"strings"

	"goc/claudemd"
	"goc/sessiontranscript"
	"goc/types"
)

// dreamPromptPrefix matches src/skills/bundled/dream.ts DREAM_PROMPT_PREFIX.
const dreamPromptPrefix = `# Dream: Memory Consolidation (manual run)

You are performing a manual dream — a reflective pass over your memory files. Unlike the automatic background dream, this run has full tool permissions and the user is watching. Synthesize what you've learned recently into durable, well-organized memories so that future sessions can orient quickly.

`

// dirExistsGuidance matches src/memdir/memdir.ts DIR_EXISTS_GUIDANCE.
const dirExistsGuidance = "This directory already exists — write to it directly with the Write tool (do not run mkdir or check for its existence)."

const dreamEntrypointName = "MEMORY.md"
const dreamMaxEntrypointLines = 200

// buildConsolidationPrompt mirrors src/services/autoDream/consolidationPrompt.ts buildConsolidationPrompt.
func buildConsolidationPrompt(memoryRoot, transcriptDir, extra string) string {
	grepExample := fmt.Sprintf("grep -rn \"<narrow term>\" %s/ --include=\"*.jsonl\" | tail -50", transcriptDir)
	s := fmt.Sprintf(`# Dream: Memory Consolidation

You are performing a dream — a reflective pass over your memory files. Synthesize what you've learned recently into durable, well-organized memories so that future sessions can orient quickly.

Memory directory: `+"`%s`"+`
%s

Session transcripts: `+"`%s`"+` (large JSONL files — grep narrowly, don't read whole files)

---

## Phase 1 — Orient

- `+"`ls`"+` the memory directory to see what already exists
- Read `+"`%s`"+` to understand the current index
- Skim existing topic files so you improve them rather than creating duplicates
- If `+"`logs/`"+` or `+"`sessions/`"+` subdirectories exist (assistant-mode layout), review recent entries there

## Phase 2 — Gather recent signal

Look for new information worth persisting. Sources in rough priority order:

1. **Daily logs** (`+"`logs/YYYY/MM/YYYY-MM-DD.md`"+`) if present — these are the append-only stream
2. **Existing memories that drifted** — facts that contradict something you see in the codebase now
3. **Transcript search** — if you need specific context (e.g., "what was the error message from yesterday's build failure?"), grep the JSONL transcripts for narrow terms:
   `+"`%s`"+`

Don't exhaustively read transcripts. Look only for things you already suspect matter.

## Phase 3 — Consolidate

For each thing worth remembering, write or update a memory file at the top level of the memory directory. Use the memory file format and type conventions from your system prompt's auto-memory section — it's the source of truth for what to save, how to structure it, and what NOT to save.

Focus on:
- Merging new signal into existing topic files rather than creating near-duplicates
- Converting relative dates ("yesterday", "last week") to absolute dates so they remain interpretable after time passes
- Deleting contradicted facts — if today's investigation disproves an old memory, fix it at the source

## Phase 4 — Prune and index

Update `+"`%s`"+` so it stays under %d lines AND under ~25KB. It's an **index**, not a dump — each entry should be one line under ~150 characters: `+"`- [Title](file.md) — one-line hook`"+`. Never write memory content directly into it.

- Remove pointers to memories that are now stale, wrong, or superseded
- Demote verbose entries: if an index line is over ~200 chars, it's carrying content that belongs in the topic file — shorten the line, move the detail
- Add pointers to newly important memories
- Resolve contradictions — if two files disagree, fix the wrong one

---

Return a brief summary of what you consolidated, updated, or pruned. If nothing changed (memories are already tight), say so.`,
		memoryRoot,
		dirExistsGuidance,
		transcriptDir,
		dreamEntrypointName,
		grepExample,
		dreamEntrypointName,
		dreamMaxEntrypointLines,
	)
	if strings.TrimSpace(extra) != "" {
		s += "\n\n## Additional context\n\n" + extra
	}
	return s
}

// resolveDream mirrors src/skills/bundled/dream.ts getPromptForCommand (recordConsolidation omitted — TS side effect).
func resolveDream(args string, opt *BundledResolveOptions) (types.SlashResolveResult, error) {
	cwd := "."
	if opt != nil && strings.TrimSpace(opt.Cwd) != "" {
		cwd = opt.Cwd
	}
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		absCwd = cwd
	}
	memoryRoot := claudemd.GetAutoMemPath(absCwd)
	transcriptDir := sessiontranscript.ProjectDirForOriginalCwd(absCwd, sessiontranscript.ConfigHomeDir())

	base := buildConsolidationPrompt(memoryRoot, transcriptDir, "")
	text := dreamPromptPrefix + base
	if s := strings.TrimSpace(args); s != "" {
		text += "\n\n## Additional context from user\n\n" + s
	}
	return types.SlashResolveResult{UserText: text, Source: types.SlashResolveBundledEmbed}, nil
}
