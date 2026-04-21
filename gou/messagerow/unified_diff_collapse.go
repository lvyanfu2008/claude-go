package messagerow

import (
	"strconv"
	"strings"
)

// DefaultUnifiedDiffContextLines mirrors TS-style `diff -U3` context: keep this many
// unchanged (space-prefixed) lines above/below each change when collapsing long runs.
const DefaultUnifiedDiffContextLines = 3

// CollapseUnifiedDiffContextLines trims long runs of context-only lines (first byte ASCII space)
// in unified diff hunks, similar to a narrow git view. Change lines (+/-) and non-space
// prefixes (e.g. "\\") are never dropped.
func CollapseUnifiedDiffContextLines(lines []string, keep int) []string {
	if keep < 1 {
		keep = DefaultUnifiedDiffContextLines
	}
	if len(lines) <= keep*2+1 {
		return append([]string(nil), lines...)
	}
	var out []string
	i := 0
	for i < len(lines) {
		ln := lines[i]
		if isUnifiedContextLine(ln) {
			start := i
			for i < len(lines) && isUnifiedContextLine(lines[i]) {
				i++
			}
			run := lines[start:i]
			if len(run) <= keep*2+1 {
				out = append(out, run...)
				continue
			}
			out = append(out, run[:keep]...)
			omitted := len(run) - 2*keep
			out = append(out, " ⋯ ("+strconv.Itoa(omitted)+" unchanged lines) ⋯")
			out = append(out, run[len(run)-keep:]...)
			continue
		}
		out = append(out, ln)
		i++
	}
	return out
}

func isUnifiedContextLine(s string) bool {
	s = strings.TrimSuffix(s, "\r")
	if s == "" {
		return false
	}
	return s[0] == ' '
}
