package toolpool

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"
)

//go:embed powershell_prompt.txt
var psPromptTemplate string

const psSleepGuidance = `
Avoid unnecessary ` + "`Start-Sleep`" + ` commands:
    - Do not sleep between commands that can run immediately — just run them.
    - If your command is long running and you would like to be notified when it finishes — simply run your command using ` + "`run_in_background`" + `. There is no need to sleep in this case.
    - Do not retry failing commands in a sleep loop — diagnose the root cause or consider an alternative approach.
    - If waiting for a background task you started with ` + "`run_in_background`" + `, you will be notified when it completes — do not poll.
    - If you must poll an external process, use a check command rather than sleeping first.
    - If you must sleep, keep the duration short (1-5 seconds) to avoid blocking the user.`

const psBackgroundNote = "\n  - You can use the `run_in_background` parameter to run the command in the background. Only use this if you don't need the result immediately and are OK being notified when the command completes later. You do not need to check the output right away - you'll be notified when it finishes."

// getPowerShellDescription mirrors PowerShellTool.getPrompt() in TS.
func getPowerShellDescription() string {
	edition := getPowerShellEdition()
	maxMs := psMaxTimeoutMs()
	defMs := psDefaultTimeoutMs()

	bgNote := "\n  - You can use the `run_in_background` parameter to run the command in the background. Only use this if you don't need the result immediately and are OK being notified when the command completes later."
	sleepGuidance := ""
	if !isPSBackgroundDisabled() {
		bgNote = psBackgroundNote
		sleepGuidance = psSleepGuidance
	}

	result := strings.ReplaceAll(psPromptTemplate, "{{BACKTICK}}", "`")
	result = strings.ReplaceAll(result, "{{EDITION_SECTION}}", getPowerShellEditionSection(edition))
	result = strings.ReplaceAll(result, "{{BG_NOTE}}", bgNote)
	result = strings.ReplaceAll(result, "{{SLEEP_GUIDANCE}}", sleepGuidance)
	result = strings.ReplaceAll(result, "{{MAX_MS}}", fmt.Sprintf("%d", maxMs))
	result = strings.ReplaceAll(result, "{{MAX_MIN}}", fmt.Sprintf("%d", maxMs/60000))
	result = strings.ReplaceAll(result, "{{DEF_MS}}", fmt.Sprintf("%d", defMs))
	result = strings.ReplaceAll(result, "{{DEF_MIN}}", fmt.Sprintf("%d", defMs/60000))

	return result
}

// psDefaultTimeoutMs returns the default PowerShell timeout in milliseconds.
func psDefaultTimeoutMs() int {
	if v := strings.TrimSpace(os.Getenv("BASH_DEFAULT_TIMEOUT_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 120_000
}

// psMaxTimeoutMs returns the max PowerShell timeout in milliseconds.
func psMaxTimeoutMs() int {
	def := psDefaultTimeoutMs()
	if v := strings.TrimSpace(os.Getenv("BASH_MAX_TIMEOUT_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > def {
				return n
			}
			return def
		}
	}
	if 600_000 > def {
		return 600_000
	}
	return def
}

// isPSBackgroundDisabled checks if background tasks are disabled for PowerShell.
func isPSBackgroundDisabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
