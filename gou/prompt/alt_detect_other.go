//go:build !windows

package prompt

func isAltKeyPressed() bool {
	return false
}
