// Mirrors src/utils/thinking.ts ThinkingConfig.
package types

// ThinkingConfig is extended thinking / ultrathink configuration for API calls.
type ThinkingConfig struct {
	Type         string `json:"type"` // adaptive | enabled | disabled
	BudgetTokens *int   `json:"budgetTokens,omitempty"`
}
