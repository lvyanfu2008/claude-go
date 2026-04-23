package toolpool

import (
	"os"
	"strings"
	"testing"
)

func TestEditToolDescription(t *testing.T) {
	// Save original env values
	originalKillswitch := os.Getenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH")
	originalUserType := os.Getenv("USER_TYPE")
	
	// Restore original env values at end of test
	defer func() {
		if originalKillswitch != "" {
			os.Setenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH", originalKillswitch)
		} else {
			os.Unsetenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH")
		}
		if originalUserType != "" {
			os.Setenv("USER_TYPE", originalUserType)
		} else {
			os.Unsetenv("USER_TYPE")
		}
	}()

	t.Run("default compact prefix", func(t *testing.T) {
		os.Unsetenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH")
		os.Unsetenv("USER_TYPE")
		
		spec := nativeEditToolSpec()
		desc := spec.Description
		
		if !strings.Contains(desc, "line number + tab") {
			t.Errorf("Expected description to contain 'line number + tab', got: %s", desc)
		}
		if !strings.Contains(desc, getPreReadInstruction()) {
			t.Errorf("Expected description to contain preread instruction")
		}
		if strings.Contains(desc, "Use the smallest old_string") {
			t.Errorf("Should not contain ant-specific uniqueness hint for non-ant user")
		}
	})

	t.Run("disabled compact prefix", func(t *testing.T) {
		os.Setenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH", "1")
		os.Unsetenv("USER_TYPE")
		
		spec := nativeEditToolSpec()
		desc := spec.Description
		
		if !strings.Contains(desc, "spaces + line number + arrow") {
			t.Errorf("Expected description to contain 'spaces + line number + arrow', got: %s", desc)
		}
	})

	t.Run("ant user type with uniqueness hint", func(t *testing.T) {
		os.Unsetenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH")
		os.Setenv("USER_TYPE", "ant")
		
		spec := nativeEditToolSpec()
		desc := spec.Description
		
		if !strings.Contains(desc, "Use the smallest old_string") {
			t.Errorf("Expected description to contain ant-specific uniqueness hint, got: %s", desc)
		}
	})
}


func TestHelperFunctions(t *testing.T) {
	t.Run("getPreReadInstruction", func(t *testing.T) {
		instruction := getPreReadInstruction()
		if !strings.Contains(instruction, "Read") {
			t.Errorf("Expected instruction to mention Read tool, got: %s", instruction)
		}
	})

	t.Run("isCompactLinePrefixEnabled", func(t *testing.T) {
		// Save original value
		original := os.Getenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH")
		defer func() {
			if original != "" {
				os.Setenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH", original)
			} else {
				os.Unsetenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH")
			}
		}()

		// Test default (enabled)
		os.Unsetenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH")
		if !isCompactLinePrefixEnabled() {
			t.Error("Expected compact prefix to be enabled by default")
		}

		// Test disabled
		os.Setenv("TENGU_COMPACT_LINE_PREFIX_KILLSWITCH", "1")
		if isCompactLinePrefixEnabled() {
			t.Error("Expected compact prefix to be disabled when killswitch is set")
		}
	})
}