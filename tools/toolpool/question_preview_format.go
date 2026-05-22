package toolpool

import "sync"

// QuestionPreviewFormat mirrors TS getQuestionPreviewFormat / setQuestionPreviewFormat
// in src/bootstrap/state.ts. Controls the preview format guidance appended to the
// AskUserQuestion tool description and enables HTML preview validation.
//
// Values: "markdown" (Go TUI default), "html" (SDK consumers), or "" (undefined — no preview).
var (
	questionPreviewFormat   string
	questionPreviewFormatMu sync.RWMutex
)

// GetQuestionPreviewFormat returns the current preview format for AskUserQuestion.
// Returns "" (undefined) when no preview format has been set, which means SDK
// consumers haven't opted into preview rendering.
func GetQuestionPreviewFormat() string {
	questionPreviewFormatMu.RLock()
	defer questionPreviewFormatMu.RUnlock()
	return questionPreviewFormat
}

// SetQuestionPreviewFormat sets the preview format for AskUserQuestion.
// Valid values: "markdown", "html", or "" (clear).
func SetQuestionPreviewFormat(format string) {
	questionPreviewFormatMu.Lock()
	defer questionPreviewFormatMu.Unlock()
	questionPreviewFormat = format
}
