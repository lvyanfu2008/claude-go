package toolpool

import (
	_ "embed"
	"strings"
)

//go:embed workflow_prompt.txt
var workflowPromptRaw string

func getWorkflowDescription() string {
	return strings.ReplaceAll(workflowPromptRaw, "{{BACKTICK}}", "`")
}
