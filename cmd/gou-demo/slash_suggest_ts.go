package main

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"goc/types"
)

// midSlashInfo mirrors commandSuggestions.ts findMidInputSlashCommand (rune-based).
type midSlashInfo struct {
	startRune int
	tokenLen  int    // in runes: "/"+partial
	partial   string // without leading '/'
}

var midSlashTailRE = regexp.MustCompile(`\s/([a-zA-Z0-9_:-]*)$`)

func isSlashCmdHidden(c *types.Command) bool {
	return c.IsHidden != nil && *c.IsHidden
}

// findMidInputSlashCommand locates a slash command token that is not at the start of
// the buffer (e.g. "help me /com"). Returns nil if not in a mid-input slash context.
func findMidInputSlashCommand(value string, cursorRune int) *midSlashInfo {
	if strings.HasPrefix(value, "/") {
		return nil
	}
	rs := []rune(value)
	if cursorRune < 0 || cursorRune > len(rs) {
		return nil
	}
	before := string(rs[:cursorRune])
	loc := midSlashTailRE.FindStringSubmatchIndex(before)
	if loc == nil {
		return nil
	}
	chunk := before[loc[0]:loc[1]]
	pSlash := strings.IndexRune(chunk, '/')
	if pSlash < 0 {
		return nil
	}
	slashByte := loc[0] + pSlash
	slashRune := utf8.RuneCountInString(before[:slashByte])

	// fullCommand: from full value after "/", until end of [a-zA-Z0-9_:-] (TS textAfterSlash)
	after := string(rs[slashRune+1:])
	var b strings.Builder
	for _, r := range after {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == ':' || r == '-' {
			b.WriteRune(r)
			continue
		}
		break
	}
	full := b.String()
	fullR := []rune(full)
	if cursorRune > slashRune+1+len(fullR) {
		return nil
	}
	return &midSlashInfo{
		startRune: slashRune,
		tokenLen:  1 + len(fullR),
		partial:   full,
	}
}

func subsequenceRunes(needle, hay string) bool {
	if needle == "" {
		return true
	}
	nb, hb := []rune(needle), []rune(hay)
	ji := 0
	for i := 0; i < len(hb) && ji < len(nb); i++ {
		if hb[i] == nb[ji] {
			ji++
		}
	}
	return ji == len(nb)
}

type slashRanked struct {
	display string
	tier    int
	fuse    float64 // lower is better; tiebreak within tier
}

// rankedSlashForQuery returns display names ("/name") ordered like TS: exact/prefix, then
// Fuse-style fuzzy and description, without adding a new dependency.
func rankedSlashForQuery(commands []types.Command, query string) []string {
	q := strings.ToLower(strings.TrimSpace(query))
	var cand []slashRanked
	for i := range commands {
		c := &commands[i]
		if isSlashCmdHidden(c) {
			continue
		}
		name := types.GetCommandName(*c)
		if name == "" {
			continue
		}
		name = strings.TrimPrefix(name, "/")
		disp := "/" + name
		nl := strings.ToLower(name)
		lowerQ := q
		if q == "" {
			cand = append(cand, slashRanked{display: disp, tier: 20, fuse: 0})
			continue
		}
		tier := 100
		var fuse float64
		// 0-4: TS-style priority; 5+ weaker matches
		switch {
		case nl == lowerQ:
			tier = 0
		case hasAliasEqual(c, lowerQ):
			tier = 1
		case strings.HasPrefix(nl, lowerQ):
			tier = 2
			fuse = float64(len(nl) - len(lowerQ))
		case hasAliasPrefix(c, lowerQ):
			tier = 3
		case strings.Contains(nl, lowerQ):
			tier = 4
			fuse = float64(strings.Index(nl, lowerQ))
		case c.Description != "" && strings.Contains(strings.ToLower(c.Description), lowerQ):
			tier = 5
			fuse = 5
		case c.WhenToUse != nil && strings.Contains(strings.ToLower(*c.WhenToUse), lowerQ):
			tier = 5
			fuse = 6
		case subsequenceRunes(q, nl):
			tier = 6
			fuse = 2
		default:
			continue
		}
		cand = append(cand, slashRanked{display: disp, tier: tier, fuse: fuse})
	}
	if len(cand) == 0 {
		return nil
	}
	sort.SliceStable(cand, func(i, j int) bool {
		if cand[i].tier != cand[j].tier {
			return cand[i].tier < cand[j].tier
		}
		if cand[i].fuse != cand[j].fuse {
			return cand[i].fuse < cand[j].fuse
		}
		return strings.ToLower(cand[i].display) < strings.ToLower(cand[j].display)
	})
	out := make([]string, 0, min(len(cand), 200))
	seen := map[string]struct{}{}
	for _, c := range cand {
		if _, ok := seen[c.display]; ok {
			continue
		}
		seen[c.display] = struct{}{}
		out = append(out, c.display)
		if len(out) >= 200 {
			break
		}
	}
	return out
}

func hasAliasEqual(c *types.Command, qLower string) bool {
	for _, a := range c.Aliases {
		if strings.ToLower(a) == qLower {
			return true
		}
	}
	return false
}

func hasAliasPrefix(c *types.Command, qLower string) bool {
	for _, a := range c.Aliases {
		al := strings.ToLower(a)
		if strings.HasPrefix(al, qLower) {
			return true
		}
	}
	return false
}

// currentSlashQuery returns the filter string and whether the primary mode is
// start-of-line slash (as opposed to mid-input). For F2-only with an empty line,
// query is "" and startMode is true.
func (m *model) currentSlashQuery() (query string, startMode bool) {
	v := m.pr.Value()
	cur := m.pr.CursorRuneIndex()
	if m.slashListUser {
		// F2: treat as browse-all, unless mid-input is active
		if mid := findMidInputSlashCommand(v, cur); mid != nil {
			return mid.partial, false
		}
		return "", true
	}
	if shouldShowTSSlashList(v, cur) {
		return slashFilterFromPrompt(v), true
	}
	if mid := findMidInputSlashCommand(v, cur); mid != nil {
		return mid.partial, false
	}
	return "", true
}

func (m *model) visibleSlashList() []string {
	m.loadSlashCommandsOnce()
	q, _ := m.currentSlashQuery()
	return rankedSlashForQuery(m.slashCommands, q)
}

func (m *model) applySlashTab() {
	if m.uiScreen != gouDemoScreenPrompt {
		return
	}
	vis := m.visibleSlashList()
	if len(vis) == 0 {
		return
	}
	if m.slashListSel < 0 || m.slashListSel >= len(vis) {
		m.slashListSel = 0
	}
	pick := vis[m.slashListSel]
	v := m.pr.Value()
	cur := m.pr.CursorRuneIndex()
	if mid := findMidInputSlashCommand(v, cur); mid != nil {
		m.replaceValueRunes(pick, mid)
	} else {
		m.pr.SetValue(strings.TrimSpace(pick) + " ")
	}
}

// replaceValueRunes replaces the mid-input slash token with pick + a trailing space, preserving
// the rest of the buffer. pick is a display name like "/compact".
func (m *model) replaceValueRunes(pick string, mid *midSlashInfo) {
	v := m.pr.Value()
	rs := []rune(v)
	if mid.startRune+mid.tokenLen > len(rs) {
		return
	}
	ins := []rune(strings.TrimSpace(pick) + " ")
	var b strings.Builder
	b.WriteString(string(rs[:mid.startRune]))
	b.WriteString(string(ins))
	b.WriteString(string(rs[mid.startRune+mid.tokenLen:]))
	m.pr.SetValue(b.String())
}
