// Port of claude-code/src/tools/shared/gitOperationTracking.ts detectGitOperation (collapse transcript only).
package messagerow

import (
	"regexp"
	"strconv"
	"strings"

	"goc/types"
)

var (
	gitCmdRe = func(subcmd, suffix string) *regexp.Regexp {
		return regexp.MustCompile(`\bgit(?:\s+-[cC]\s+\S+|\s+--\S+=\S+)*\s+` + subcmd + `\b` + suffix)
	}
	gitCommitRE    = gitCmdRe("commit", "")
	gitPushRE      = gitCmdRe("push", "")
	gitCherryPickRE = gitCmdRe("cherry-pick", "")
	// Go regexp has no lookahead; TS uses (?!-) after merge — use plain merge match.
	gitMergeRE = gitCmdRe("merge", "")
	gitRebaseRE    = gitCmdRe("rebase", "")
	gitPrActions   = []struct {
		re     *regexp.Regexp
		action types.PrAction
	}{
		{regexp.MustCompile(`\bgh\s+pr\s+create\b`), types.PrActionCreated},
		{regexp.MustCompile(`\bgh\s+pr\s+edit\b`), types.PrActionEdited},
		{regexp.MustCompile(`\bgh\s+pr\s+merge\b`), types.PrActionMerged},
		{regexp.MustCompile(`\bgh\s+pr\s+comment\b`), types.PrActionCommented},
		{regexp.MustCompile(`\bgh\s+pr\s+close\b`), types.PrActionClosed},
		{regexp.MustCompile(`\bgh\s+pr\s+ready\b`), types.PrActionReady},
	}
	parseGitCommitIDRe = regexp.MustCompile(`\[[\w./-]+(?: \(root-commit\))? ([0-9a-f]+)\]`)
	parseGitPushBranchRe = regexp.MustCompile(`(?m)^\s*[+\-*!= ]?\s*(?:\[new branch\]|\S+\.\.+\S+)\s+\S+\s*->\s*(\S+)`)
	findPrURLRe        = regexp.MustCompile(`https://github\.com/[^/\s]+/[^/\s]+/pull/\d+`)
	parsePrNumFromText = regexp.MustCompile(`[Pp]ull request (?:\S+#)?#?(\d+)`)
)

func parseGitCommitID(stdout string) string {
	m := parseGitCommitIDRe.FindStringSubmatch(stdout)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func parseGitPushBranch(output string) string {
	m := parseGitPushBranchRe.FindStringSubmatch(output)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func parsePrURL(url string) (number int, prURL string, ok bool) {
	m := regexp.MustCompile(`https://github\.com/([^/]+/[^/]+)/pull/(\d+)`).FindStringSubmatch(url)
	if len(m) < 3 {
		return 0, "", false
	}
	n, err := strconv.Atoi(m[2])
	if err != nil {
		return 0, "", false
	}
	return n, url, true
}

func findPrInStdout(stdout string) (number int, url string, ok bool) {
	m := findPrURLRe.FindString(stdout)
	if m == "" {
		return 0, "", false
	}
	n, u, ok := parsePrURL(m)
	return n, u, ok
}

func parsePrNumberFromText(stdout string) (int, bool) {
	m := parsePrNumFromText.FindStringSubmatch(stdout)
	if len(m) < 2 {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

func parseRefFromCommand(command, verb string) string {
	re := gitCmdRe(verb, "")
	idx := re.FindStringIndex(command)
	if idx == nil {
		return ""
	}
	after := strings.TrimSpace(command[idx[1]:])
	for _, t := range strings.Fields(after) {
		if len(t) > 0 && strings.ContainsAny(string(t[0]), "&|;><") {
			break
		}
		if strings.HasPrefix(t, "-") {
			continue
		}
		return t
	}
	return ""
}

// detectGitOperationGo mirrors TS detectGitOperation.
func detectGitOperationGo(command, output string) (commit *types.GitCommitEntry, push *types.GitPushEntry, branch *types.GitBranchEntry, pr *types.GitPrEntry) {
	combined := output
	isCherry := gitCherryPickRE.MatchString(command)
	if gitCommitRE.MatchString(command) || isCherry {
		sha := parseGitCommitID(combined)
		if sha != "" {
			kind := types.CommitKindCommitted
			if isCherry {
				kind = types.CommitKindCherryPicked
			} else if regexp.MustCompile(`--amend\b`).MatchString(command) {
				kind = types.CommitKindAmended
			}
			s6 := sha
			if len(s6) > 6 {
				s6 = s6[:6]
			}
			commit = &types.GitCommitEntry{Sha: s6, Kind: kind}
		}
	}
	if gitPushRE.MatchString(command) {
		b := parseGitPushBranch(combined)
		if b != "" {
			push = &types.GitPushEntry{Branch: b}
		}
	}
	if gitMergeRE.MatchString(command) && (strings.Contains(combined, "Fast-forward") || strings.Contains(combined, "Merge made by")) {
		ref := parseRefFromCommand(command, "merge")
		if ref != "" {
			branch = &types.GitBranchEntry{Ref: ref, Action: types.BranchActionMerged}
		}
	}
	if gitRebaseRE.MatchString(command) && strings.Contains(combined, "Successfully rebased") {
		ref := parseRefFromCommand(command, "rebase")
		if ref != "" {
			branch = &types.GitBranchEntry{Ref: ref, Action: types.BranchActionRebased}
		}
	}
	for _, row := range gitPrActions {
		if !row.re.MatchString(command) {
			continue
		}
		if n, u, ok := findPrInStdout(combined); ok {
			url := u
			pr = &types.GitPrEntry{Number: n, URL: &url, Action: row.action}
		} else if n, ok := parsePrNumberFromText(combined); ok {
			pr = &types.GitPrEntry{Number: n, Action: row.action}
		}
		break
	}
	return commit, push, branch, pr
}
