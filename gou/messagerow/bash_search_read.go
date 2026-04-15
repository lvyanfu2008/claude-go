// Mirrors claude-code/src/tools/BashTool/BashTool.tsx isSearchOrReadBashCommand
// (BASH_* sets + pipeline split). Used for collapsed_read_search tail rollup.
package messagerow

import (
	"regexp"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Matches backslash-newline sequences (TS splitCommandWithOperators continuation join).
var bashContinuationRegexp = regexp.MustCompile(`\\+\n`)

// Bash command sets (BashTool.tsx BASH_SEARCH_COMMANDS / BASH_READ_COMMANDS / BASH_LIST_COMMANDS / BASH_SEMANTIC_NEUTRAL_COMMANDS).
var (
	bashSearchCommands = map[string]struct{}{
		"find": {}, "grep": {}, "rg": {}, "ag": {}, "ack": {}, "locate": {}, "which": {}, "whereis": {},
	}
	bashReadCommands = map[string]struct{}{
		"cat": {}, "head": {}, "tail": {}, "less": {}, "more": {},
		"wc": {}, "stat": {}, "file": {}, "strings": {},
		"jq": {}, "awk": {}, "cut": {}, "sort": {}, "uniq": {}, "tr": {},
	}
	bashListCommands = map[string]struct{}{
		"ls": {}, "tree": {}, "du": {},
	}
	bashSemanticNeutralCommands = map[string]struct{}{
		"echo": {}, "printf": {}, "true": {}, "false": {}, ":": {},
	}
)

// joinBashContinuationLines mirrors TS splitCommandWithOperators backslash-newline join (odd count of \ before \n).
func joinBashContinuationLines(command string) string {
	return bashContinuationRegexp.ReplaceAllStringFunc(command, func(match string) string {
		backslashCount := len(match) - 1
		if backslashCount%2 == 1 {
			return strings.Repeat(`\`, backslashCount-1)
		}
		return match
	})
}

// IsSearchOrReadBashCommand returns whether a bash command is classified as search/read/list per TS.
// On parse failure, the whole string is treated as one part (TS splitCommandWithOperators catch path).
func IsSearchOrReadBashCommand(command string) (isSearch, isRead, isList bool) {
	cmd := strings.TrimSpace(joinBashContinuationLines(command))
	if cmd == "" {
		return false, false, false
	}
	p := syntax.NewParser()
	f, err := p.Parse(strings.NewReader(cmd), "")
	if err != nil {
		return classifyBashFirstWords([]string{firstShellWord(cmd)})
	}
	var words []string
	for _, st := range f.Stmts {
		ws := leafFirstWords(st)
		if len(ws) == 0 {
			return classifyBashFirstWords([]string{firstShellWord(cmd)})
		}
		words = append(words, ws...)
	}
	if len(words) == 0 {
		return classifyBashFirstWords([]string{firstShellWord(cmd)})
	}
	return classifyBashFirstWords(words)
}

func firstShellWord(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return strings.Fields(s)[0]
}

func classifyBashFirstWords(parts []string) (isSearch, isRead, isList bool) {
	var hasSearch, hasRead, hasList bool
	hasNonNeutral := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		base := strings.Fields(part)[0]
		if _, ok := bashSemanticNeutralCommands[base]; ok {
			continue
		}
		hasNonNeutral = true
		_, isS := bashSearchCommands[base]
		_, isR := bashReadCommands[base]
		_, isL := bashListCommands[base]
		if !isS && !isR && !isL {
			return false, false, false
		}
		if isS {
			hasSearch = true
		}
		if isR {
			hasRead = true
		}
		if isL {
			hasList = true
		}
	}
	if !hasNonNeutral {
		return false, false, false
	}
	return hasSearch, hasRead, hasList
}

// leafFirstWords returns the first word of each leaf simple command in a pipeline / && / || chain.
func leafFirstWords(s *syntax.Stmt) []string {
	if s == nil || s.Cmd == nil {
		return nil
	}
	switch c := s.Cmd.(type) {
	case *syntax.BinaryCmd:
		switch c.Op {
		case syntax.AndStmt, syntax.OrStmt, syntax.Pipe, syntax.PipeAll:
			left := leafFirstWords(c.X)
			right := leafFirstWords(c.Y)
			if len(left) == 0 && len(right) == 0 {
				return nil
			}
			return append(left, right...)
		}
		return nil
	case *syntax.CallExpr:
		w := firstWordCallExpr(c)
		if w == "" {
			return nil
		}
		return []string{w}
	default:
		return nil
	}
}

func firstWordCallExpr(c *syntax.CallExpr) string {
	for _, arg := range c.Args {
		if w := wordFirstField(arg); w != "" {
			return w
		}
	}
	return ""
}

func wordFirstField(w *syntax.Word) string {
	if w == nil {
		return ""
	}
	if lit := w.Lit(); lit != "" {
		fs := strings.Fields(strings.TrimSpace(lit))
		if len(fs) > 0 {
			return fs[0]
		}
	}
	return ""
}
