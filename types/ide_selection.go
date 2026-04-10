// Mirrors src/hooks/useIdeSelection.ts IDESelection.
package types

// IDESelection is the IDE-reported selection summary for @-attachments / context.
type IDESelection struct {
	LineCount int     `json:"lineCount"`
	LineStart *int    `json:"lineStart,omitempty"`
	Text      *string `json:"text,omitempty"`
	FilePath  *string `json:"filePath,omitempty"`
}
