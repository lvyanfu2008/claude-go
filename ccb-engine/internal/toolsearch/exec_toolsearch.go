package toolsearch

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"goc/ccb-engine/internal/anthropic"
)

// ExecToolSearchForRunner mirrors ToolSearchTool.call + mapToolResultToToolResultBlockParam (src/tools/ToolSearchTool/ToolSearchTool.ts):
//   - select: — resolves each name against the full registry (deferred or already loaded), like findToolByName(deferred) ?? findToolByName(tools)
//   - keyword — exact name, mcp__ prefix scan, +required terms, scoring on name parts + description (static; TS uses tool.prompt())
//   - empty matches — plain text like TS (optional MCP connecting suffix when hasPendingMcpServers)
//   - non-empty — JSON array of {"type":"tool_reference","tool_name":...} as tool_result string (TS content array shape)
func ExecToolSearchForRunner(input json.RawMessage, allTools []anthropic.ToolDefinition, hasPendingMcpServers bool, pendingMcpServerNames []string) (string, bool, error) {
	var in struct {
		Query      string  `json:"query"`
		MaxResults float64 `json:"max_results"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", true, err
	}
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return "", true, nil
	}
	max := int(in.MaxResults)
	if max <= 0 {
		max = 5
	}
	var matches []string
	if len(allTools) > 0 {
		deferred := deferredFromTools(allTools)
		if isSelect, sel := parseSelectQuery(q, allTools, deferred, max); isSelect {
			matches = sel
		} else {
			matches = searchToolsWithKeywords(q, deferred, allTools, max)
		}
	} else {
		matches = resolveToolSearchQueryBuiltinOnly(q, max)
	}
	if len(matches) == 0 {
		return toolSearchEmptyMessage(hasPendingMcpServers, pendingMcpServerNames), false, nil
	}
	refs := make([]map[string]any, 0, len(matches))
	for _, n := range matches {
		refs = append(refs, map[string]any{"type": "tool_reference", "tool_name": n})
	}
	b, err := json.Marshal(refs)
	if err != nil {
		return "", true, err
	}
	return string(b), false, nil
}

func deferredFromTools(all []anthropic.ToolDefinition) []anthropic.ToolDefinition {
	var out []anthropic.ToolDefinition
	for i := range all {
		if IsDeferredToolName(all[i].Name) {
			out = append(out, all[i])
		}
	}
	return out
}

func findToolByNameCI(all []anthropic.ToolDefinition, want string) (anthropic.ToolDefinition, bool) {
	want = strings.TrimSpace(strings.ToLower(want))
	for i := range all {
		if strings.ToLower(strings.TrimSpace(all[i].Name)) == want {
			return all[i], true
		}
	}
	return anthropic.ToolDefinition{}, false
}

var (
	selectPrefixRE = regexp.MustCompile(`(?i)^select:(.+)$`)
	camelSplitRE   = regexp.MustCompile(`([a-z])([A-Z])`)
)

// parseSelectQuery returns (true, names) when query is select:... (even if names is empty); (false, nil) otherwise.
func parseSelectQuery(q string, allTools, deferred []anthropic.ToolDefinition, max int) (bool, []string) {
	m := selectPrefixRE.FindStringSubmatch(q)
	if m == nil {
		return false, nil
	}
	tail := strings.TrimSpace(m[1])
	if tail == "" {
		return true, nil
	}
	var found []string
	for _, part := range strings.Split(tail, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if t, ok := findToolByNameCI(deferred, name); ok {
			found = appendUnique(found, t.Name)
		} else if t, ok := findToolByNameCI(allTools, name); ok {
			found = appendUnique(found, t.Name)
		}
		if len(found) >= max {
			break
		}
	}
	return true, found
}

func appendUnique(xs []string, v string) []string {
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}

func searchToolsWithKeywords(query string, deferredTools, allTools []anthropic.ToolDefinition, maxResults int) []string {
	queryLower := strings.ToLower(strings.TrimSpace(query))
	if t, ok := findToolByNameCI(deferredTools, queryLower); ok {
		return []string{t.Name}
	}
	if t, ok := findToolByNameCI(allTools, queryLower); ok {
		return []string{t.Name}
	}
	if strings.HasPrefix(queryLower, "mcp__") && len(queryLower) > 5 {
		var prefixMatches []string
		for i := range deferredTools {
			n := deferredTools[i].Name
			if strings.HasPrefix(strings.ToLower(n), queryLower) {
				prefixMatches = append(prefixMatches, n)
			}
		}
		if len(prefixMatches) > 0 {
			sort.Strings(prefixMatches)
			if len(prefixMatches) > maxResults {
				prefixMatches = prefixMatches[:maxResults]
			}
			return prefixMatches
		}
	}
	queryTerms := strings.Fields(queryLower)
	if len(queryTerms) == 0 {
		return nil
	}
	var requiredTerms, optionalTerms []string
	for _, term := range queryTerms {
		if strings.HasPrefix(term, "+") && len(term) > 1 {
			requiredTerms = append(requiredTerms, term[1:])
		} else {
			optionalTerms = append(optionalTerms, term)
		}
	}
	allScoringTerms := queryTerms
	if len(requiredTerms) > 0 {
		allScoringTerms = append(append([]string{}, requiredTerms...), optionalTerms...)
	}
	patterns := compileTermPatterns(allScoringTerms)

	candidates := deferredTools
	if len(requiredTerms) > 0 {
		var filtered []anthropic.ToolDefinition
		for i := range deferredTools {
			t := deferredTools[i]
			if toolMatchesAllRequired(t, requiredTerms, patterns) {
				filtered = append(filtered, t)
			}
		}
		candidates = filtered
	}

	type scored struct {
		name  string
		score int
	}
	var hits []scored
	for i := range candidates {
		t := candidates[i]
		parts, full, isMcp := parseToolName(t.Name)
		descNorm := strings.ToLower(t.Description)
		sc := scoreTool(parts, full, isMcp, descNorm, "", allScoringTerms, patterns)
		if sc > 0 {
			hits = append(hits, scored{t.Name, sc})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].name < hits[j].name
	})
	out := make([]string, 0, maxResults)
	for _, h := range hits {
		if len(out) >= maxResults {
			break
		}
		out = append(out, h.name)
	}
	return out
}

func compileTermPatterns(terms []string) map[string]*regexp.Regexp {
	m := make(map[string]*regexp.Regexp)
	for _, term := range terms {
		if term == "" {
			continue
		}
		if _, ok := m[term]; ok {
			continue
		}
		m[term] = regexp.MustCompile(`\b` + regexp.QuoteMeta(term) + `\b`)
	}
	return m
}

func toolMatchesAllRequired(t anthropic.ToolDefinition, required []string, patterns map[string]*regexp.Regexp) bool {
	parts, full, _ := parseToolName(t.Name)
	descNorm := strings.ToLower(t.Description)
	for _, term := range required {
		pat := patterns[term]
		if pat == nil {
			continue
		}
		ok := false
		for _, p := range parts {
			if p == term || strings.Contains(p, term) {
				ok = true
				break
			}
		}
		if !ok && strings.Contains(full, term) {
			ok = true
		}
		if !ok && pat.MatchString(descNorm) {
			ok = true
		}
		if !ok {
			return false
		}
	}
	return true
}

func scoreTool(parts []string, full string, isMcp bool, descNorm, hintNorm string, allTerms []string, patterns map[string]*regexp.Regexp) int {
	score := 0
	for _, term := range allTerms {
		if term == "" {
			continue
		}
		pat := patterns[term]
		termScore := 0
		partHit := false
		for _, p := range parts {
			if p == term {
				termScore += pick(isMcp, 12, 10)
				partHit = true
				break
			}
			if strings.Contains(p, term) {
				termScore += pick(isMcp, 6, 5)
				partHit = true
				break
			}
		}
		if !partHit && strings.Contains(full, term) {
			termScore += 3
		}
		if pat != nil {
			if hintNorm != "" && pat.MatchString(hintNorm) {
				termScore += 4
			}
			if pat.MatchString(descNorm) {
				termScore += 2
			}
		}
		score += termScore
	}
	return score
}

func pick(isMcp bool, mcp, reg int) int {
	if isMcp {
		return mcp
	}
	return reg
}

// parseToolName mirrors ToolSearchTool.ts parseToolName (MCP vs CamelCase).
func parseToolName(name string) (parts []string, full string, isMcp bool) {
	if strings.HasPrefix(name, "mcp__") {
		without := strings.ToLower(strings.TrimPrefix(name, "mcp__"))
		var flat []string
		for _, seg := range strings.Split(without, "__") {
			for _, p := range strings.Split(seg, "_") {
				p = strings.TrimSpace(p)
				if p != "" {
					flat = append(flat, p)
				}
			}
		}
		full = strings.ReplaceAll(strings.ReplaceAll(without, "__", " "), "_", " ")
		return flat, full, true
	}
	s := camelSplitRE.ReplaceAllString(name, "$1 $2")
	s = strings.ReplaceAll(s, "_", " ")
	parts = strings.Fields(strings.ToLower(s))
	full = strings.Join(parts, " ")
	return parts, full, false
}

func resolveToolSearchQueryBuiltinOnly(q string, max int) []string {
	lower := strings.ToLower(strings.TrimSpace(q))
	if strings.HasPrefix(lower, "select:") {
		idx := strings.Index(q, ":")
		tail := strings.TrimSpace(q[idx+1:])
		var out []string
		for _, part := range strings.Split(tail, ",") {
			name := strings.TrimSpace(part)
			if name == "" || !IsDeferredToolName(name) {
				continue
			}
			out = appendUnique(out, name)
			if len(out) >= max {
				break
			}
		}
		return out
	}
	keywords := strings.Fields(lower)
	if len(keywords) == 0 {
		return nil
	}
	var names []string
	for n := range deferredBuiltin {
		names = append(names, n)
	}
	sort.Strings(names)
	type scored struct {
		name  string
		score int
	}
	var hits []scored
	for _, name := range names {
		lname := strings.ToLower(name)
		sc := 0
		for _, kw := range keywords {
			if kw == "" {
				continue
			}
			if strings.Contains(lname, kw) {
				sc++
			}
		}
		if sc > 0 {
			hits = append(hits, scored{name, sc})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].name < hits[j].name
	})
	out := make([]string, 0, max)
	for _, h := range hits {
		if len(out) >= max {
			break
		}
		out = append(out, h.name)
	}
	return out
}

func toolSearchEmptyMessage(hasPendingMcp bool, pendingNames []string) string {
	msg := "No matching deferred tools found"
	if !hasPendingMcp {
		return msg
	}
	if len(pendingNames) > 0 {
		return msg + ". Some MCP servers are still connecting: " + strings.Join(pendingNames, ", ") + ". Their tools will become available shortly — try searching again."
	}
	return msg + ". Some MCP servers are still connecting. Their tools will become available shortly — try searching again."
}
