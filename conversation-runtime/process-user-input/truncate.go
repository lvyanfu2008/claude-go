package processuserinput

import "fmt"

const maxHookOutputLength = 10000

func applyTruncation(content string) string {
	if len(content) <= maxHookOutputLength {
		return content
	}
	return content[:maxHookOutputLength] + fmt.Sprintf("… [output truncated - exceeded %d characters]", maxHookOutputLength)
}
