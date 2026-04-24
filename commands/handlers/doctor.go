package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
)

// DoctorResult is the JSON payload returned by /doctor.
type DoctorResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleDoctorCommand runs basic diagnostics and returns the result as text.
// Mirrors TS src/commands/doctor/ (local-jsx -> Doctor component).
func HandleDoctorCommand() ([]byte, error) {
	version := "gou-demo (dev)"
	if bi, ok := debug.ReadBuildInfo(); ok {
		v := bi.Main.Version
		if v != "" && v != "(devel)" {
			version = v
		}
	}

	cwd, _ := os.Getwd()

	lines := []string{
		"Claude Code (gou-demo) — Doctor Report",
		"══════════════════════════════════════",
		fmt.Sprintf("  Version:     %s", version),
		fmt.Sprintf("  OS:          %s / %s", runtime.GOOS, runtime.GOARCH),
		fmt.Sprintf("  CWD:         %s", cwd),
		fmt.Sprintf("  Go version:  %s", runtime.Version()),
	}

	// Check CLAUDE.md
	if _, err := os.Stat(cwd + "/CLAUDE.md"); err == nil {
		lines = append(lines, "  CLAUDE.md:   ✓ Found")
	} else {
		lines = append(lines, "  CLAUDE.md:   ✗ Not found (run /init)")
	}

	// Check .claude directory
	if fi, err := os.Stat(cwd + "/.claude"); err == nil && fi.IsDir() {
		lines = append(lines, "  .claude/:    ✓ Found")
	} else {
		lines = append(lines, "  .claude/:    ✗ Not found")
	}

	// Check git
	if _, err := os.Stat(cwd + "/.git"); err == nil {
		lines = append(lines, "  Git repo:    ✓ Yes")
	} else {
		lines = append(lines, "  Git repo:    ✗ No")
	}

	lines = append(lines, "")
	lines = append(lines, "For a full diagnosis, use the TS CLI: claude /doctor")

	msg := DoctorResult{
		Type:  "text",
		Value: joinLines(lines),
	}
	return json.Marshal(msg)
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}
