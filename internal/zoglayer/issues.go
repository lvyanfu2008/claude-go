package zoglayer

import (
	"fmt"

	z "github.com/Oudwins/zog"
)

func issuesToErr(issues z.ZogIssueList) error {
	if issues == nil || len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("zog: %v", issues)
}
