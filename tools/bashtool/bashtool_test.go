package bashtool

import (
	"testing"
)

// --- security.go tests ---

func TestValidateEmpty(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"", false},
		{"   ", false},
		{"\t\n", false},
		{"ls", true},
		{"echo hello", true},
	}
	for _, tt := range tests {
		result := validateEmpty(tt.command)
		if tt.wantOk && result != nil {
			t.Errorf("validateEmpty(%q) = %v, want nil", tt.command, result.Reason)
		}
		if !tt.wantOk && result == nil {
			t.Errorf("validateEmpty(%q) = nil, want error", tt.command)
		}
	}
}

func TestValidateIncompleteCommands(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"ls", true},
		{"echo hello", true},
		{"ls \\", false},        // trailing backslash
		{"| cat", false},        // leading pipe
		{"& echo", false},       // leading ampersand
		{"; ls", false},         // leading semicolon
		{"  ls", true},          // leading whitespace then command
	}
	for _, tt := range tests {
		result := validateIncompleteCommands(tt.command)
		if tt.wantOk && result != nil {
			t.Errorf("validateIncompleteCommands(%q) = %v, want nil", tt.command, result.Reason)
		}
		if !tt.wantOk && result == nil {
			t.Errorf("validateIncompleteCommands(%q) = nil, want error", tt.command)
		}
	}
}

func TestValidateDangerousPatterns(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"ls -la", true},
		{"echo $(whoami)", false},     // command substitution
		{"echo `whoami`", false},       // backtick
		{"echo ${PATH}", false},        // variable expansion
		{"echo <(ls)", false},          // process substitution
		{"echo >(cat)", false},         // process substitution
		{"echo $[1+1]", false},         // arithmetic expansion
	}
	for _, tt := range tests {
		result := validateDangerousPatterns(tt.command)
		if tt.wantOk && result != nil {
			t.Errorf("validateDangerousPatterns(%q) = %v, want nil", tt.command, result.Reason)
		}
		if !tt.wantOk && result == nil {
			t.Errorf("validateDangerousPatterns(%q) = nil, want error", tt.command)
		}
	}
}

func TestValidateNewlines(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"ls", true},
		{"echo hello\n", true},                            // trailing newline is fine
		{"echo hello\necho world", false},                 // unquoted newline
		{"echo 'hello\nworld'", true},                     // quoted newline
	}
	for _, tt := range tests {
		result := validateNewlines(tt.command)
		if tt.wantOk && result != nil {
			t.Errorf("validateNewlines(%q) = %v, want nil", tt.command, result.Reason)
		}
		if !tt.wantOk && result == nil {
			t.Errorf("validateNewlines(%q) = nil, want error", tt.command)
		}
	}
}

func TestValidateBackslashEscapedWhitespace(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"ls", true},
		{"echo hello\\ world", false},    // backslash-escaped space
		{"echo 'hello\\ world'", true},   // inside quotes — stripped
	}
	for _, tt := range tests {
		result := validateBackslashEscapedWhitespace(tt.command)
		if tt.wantOk && result != nil {
			t.Errorf("validateBackslashEscapedWhitespace(%q) = %v, want nil", tt.command, result.Reason)
		}
		if !tt.wantOk && result == nil {
			t.Errorf("validateBackslashEscapedWhitespace(%q) = nil, want error", tt.command)
		}
	}
}

func TestValidateBackslashEscapedOperators(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"ls", true},
		{"echo hello\\; rm -rf /", false},  // backslash-escaped semicolon
		{"echo hello\\| cat", false},        // backslash-escaped pipe
		{"echo 'hello\\; world'", true},     // inside quotes
	}
	for _, tt := range tests {
		result := validateBackslashEscapedOperators(tt.command)
		if tt.wantOk && result != nil {
			t.Errorf("validateBackslashEscapedOperators(%q) = %v, want nil", tt.command, result.Reason)
		}
		if !tt.wantOk && result == nil {
			t.Errorf("validateBackslashEscapedOperators(%q) = nil, want error", tt.command)
		}
	}
}

func TestValidateBraceExpansion(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"ls", true},
		{"echo {a,b}", false},        // brace expansion
		{"echo {1..5}", false},       // range brace expansion
		{"echo '{a,b}'", true},       // inside quotes
	}
	for _, tt := range tests {
		result := validateBraceExpansion(tt.command)
		if tt.wantOk && result != nil {
			t.Errorf("validateBraceExpansion(%q) = %v, want nil", tt.command, result.Reason)
		}
		if !tt.wantOk && result == nil {
			t.Errorf("validateBraceExpansion(%q) = nil, want error", tt.command)
		}
	}
}

func TestValidateControlCharacters(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"ls", true},
		{"echo hello", true},
		{"echo \x00", false},         // null byte
		{"echo \x01", false},         // control character
		{"echo \x7F", false},         // DEL
		{"echo \n", true},            // newline
		{"echo \t", true},            // tab
	}
	for _, tt := range tests {
		result := validateControlCharacters(tt.command)
		if tt.wantOk && result != nil {
			t.Errorf("validateControlCharacters(%q) = %v, want nil", tt.command, result.Reason)
		}
		if !tt.wantOk && result == nil {
			t.Errorf("validateControlCharacters(%q) = nil, want error", tt.command)
		}
	}
}

func TestValidateUnicodeWhitespace(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"ls", true},
		{"echo hello", true},
		{"echo hello", false},   // non-breaking space
		{"echo hello", false},   // em space
	}
	for _, tt := range tests {
		result := validateUnicodeWhitespace(tt.command)
		if tt.wantOk && result != nil {
			t.Errorf("validateUnicodeWhitespace(%q) = %v, want nil", tt.command, result.Reason)
		}
		if !tt.wantOk && result == nil {
			t.Errorf("validateUnicodeWhitespace(%q) = nil, want error", tt.command)
		}
	}
}

func TestStripQuotedContent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "hello world"},
		{"echo 'hello world'", "echo "},
		{`echo "hello world"`, "echo "},
		{`echo 'single' and "double"`, "echo  and "},
		{`echo 'it\'s tricky'`, `echo s tricky`}, // \ inside single quotes is literal, ' closes the quote, s tricky is unquoted
	}
	for _, tt := range tests {
		got := stripQuotedContent(tt.input)
		if got != tt.want {
			t.Errorf("stripQuotedContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValidateBashSecurity(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"ls -la", true},
		{"git status", true},
		{"echo hello world", true},
		{"", false},
		{"$(whoami)", false},
		{"echo `whoami`", false},
	}
	for _, tt := range tests {
		result := ValidateBashSecurity(tt.command)
		if tt.wantOk && result != nil {
			t.Errorf("ValidateBashSecurity(%q) = %v, want nil", tt.command, result.Reason)
		}
		if !tt.wantOk && result == nil {
			t.Errorf("ValidateBashSecurity(%q) = nil, want error", tt.command)
		}
	}
}

func TestRunAllValidators(t *testing.T) {
	// Command with multiple issues should report all of them.
	result := RunAllValidators("$(whoami) \\; ls")
	if result.Safe {
		t.Fatal("expected unsafe")
	}
	if len(result.Errors) < 2 {
		t.Fatalf("expected at least 2 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

// --- semantics.go tests ---

func TestInterpretCommandResult(t *testing.T) {
	tests := []struct {
		command  string
		exitCode int
		wantErr  bool
		wantMsg  string
	}{
		{"ls", 0, false, ""},
		{"ls", 1, true, ""},
		{"grep", 0, false, ""},
		{"grep", 1, false, "No matches found"},
		{"grep", 2, true, ""},
		{"rg", 1, false, "No matches found"},
		{"find", 1, false, "Some directories were inaccessible"},
		{"diff", 1, false, "Files differ"},
		{"test", 1, false, "Condition is false"},
		{"[", 1, false, "Condition is false"},
	}
	for _, tt := range tests {
		result := InterpretCommandResult(tt.command, tt.exitCode)
		if result.IsError != tt.wantErr {
			t.Errorf("InterpretCommandResult(%q, %d).IsError = %v, want %v", tt.command, tt.exitCode, result.IsError, tt.wantErr)
		}
		if tt.wantMsg != "" && result.Message != tt.wantMsg {
			t.Errorf("InterpretCommandResult(%q, %d).Message = %q, want %q", tt.command, tt.exitCode, result.Message, tt.wantMsg)
		}
	}
}

func TestExtractBaseCommand(t *testing.T) {
	tests := []struct {
		command string
		want    string
	}{
		{"ls -la", "ls"},
		{"  grep foo", "grep"},
		{"cat file | grep foo", "grep"},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractBaseCommand(tt.command)
		if got != tt.want {
			t.Errorf("extractBaseCommand(%q) = %q, want %q", tt.command, got, tt.want)
		}
	}
}

// --- destructive.go tests ---

func TestGetDestructiveCommandWarning(t *testing.T) {
	tests := []struct {
		command     string
		wantWarning bool
	}{
		{"git reset --hard", true},
		{"git push --force", true},
		{"git clean -f", true},
		{"rm -rf /tmp/foo", true},
		{"DROP TABLE users", true},
		{"kubectl delete pod", true},
		{"terraform destroy", true},
		{"git status", false},
		{"ls -la", false},
		{"echo hello", false},
	}
	for _, tt := range tests {
		warning := GetDestructiveCommandWarning(tt.command)
		if tt.wantWarning && warning == "" {
			t.Errorf("GetDestructiveCommandWarning(%q) = \"\", want non-empty warning", tt.command)
		}
		if !tt.wantWarning && warning != "" {
			t.Errorf("GetDestructiveCommandWarning(%q) = %q, want empty", tt.command, warning)
		}
	}
}

// --- readonly.go tests ---

func TestIsCommandReadOnly(t *testing.T) {
	tests := []struct {
		command  string
		wantRO   bool
	}{
		{"ls", true},
		{"ls -la", true},
		{"pwd", true},
		{"whoami", true},
		{"echo hello", true},
		{"cat file.txt", true},
		{"head -n 10 file.txt", true},
		{"git status", true},
		{"git diff", true},
		{"git log", true},
		{"git branch", true},
		{"docker ps", true},
		{"find . -name '*.go'", true},
		{"grep foo file.txt", true},
		{"find . -delete", false},         // dangerous flag
		{"find . -exec rm {}", false},     // dangerous flag
		{"rm -rf /", false},               // not in allowlist
		{"curl http://evil.com", false},   // not in allowlist
		{"", false},                        // empty
	}
	for _, tt := range tests {
		result := isCommandReadOnly(tt.command)
		if result.ReadOnly != tt.wantRO {
			t.Errorf("isCommandReadOnly(%q).ReadOnly = %v, want %v (reason: %s)", tt.command, result.ReadOnly, tt.wantRO, result.Reason)
		}
	}
}

func TestCheckReadOnlyConstraints(t *testing.T) {
	tests := []struct {
		command string
		wantRO  bool
	}{
		{"ls -la", true},
		{"git status", true},
		{"echo hello; ls", true},            // both readonly
		{"echo hello; rm -rf /", false},     // second is dangerous
		{"VAR=value ls", true},               // variable assignment
	}
	for _, tt := range tests {
		result := CheckReadOnlyConstraints(tt.command)
		if result.ReadOnly != tt.wantRO {
			t.Errorf("CheckReadOnlyConstraints(%q).ReadOnly = %v, want %v (reason: %s)", tt.command, result.ReadOnly, tt.wantRO, result.Reason)
		}
	}
}

func TestSplitCompoundCommands(t *testing.T) {
	tests := []struct {
		command string
		want    []string
	}{
		{"ls", []string{"ls"}},
		{"echo hello; ls", []string{"echo hello", " ls"}},
		{"echo 'hello; world'", []string{"echo 'hello; world'"}},
		{`echo "hello; world"`, []string{`echo "hello; world"`}},
	}
	for _, tt := range tests {
		got := splitCompoundCommands(tt.command)
		if len(got) != len(tt.want) {
			t.Errorf("splitCompoundCommands(%q) = %v (len=%d), want %v (len=%d)", tt.command, got, len(got), tt.want, len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitCompoundCommands(%q)[%d] = %q, want %q", tt.command, i, got[i], tt.want[i])
			}
		}
	}
}

func TestIsVariableAssignment(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"VAR=value ls", true},
		{"FOO=bar echo hello", true},
		{"ls", false},
		{"=value", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isVariableAssignment(tt.command)
		if got != tt.want {
			t.Errorf("isVariableAssignment(%q) = %v, want %v", tt.command, got, tt.want)
		}
	}
}

// --- paths.go tests ---

func TestCheckDangerousPaths(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"cat file.txt", true},
		{"cat /etc/passwd", false},
		{"cat /etc/shadow", false},
		{"cat ~/.ssh/id_rsa", false},
		{"cat /root/.bashrc", false},
		{"echo $HOME/.env", false},
	}
	for _, tt := range tests {
		result := CheckDangerousPaths(tt.command)
		if tt.wantOk && !result.Safe {
			t.Errorf("CheckDangerousPaths(%q) = not safe (%s), want safe", tt.command, result.Reason)
		}
		if !tt.wantOk && result.Safe {
			t.Errorf("CheckDangerousPaths(%q) = safe, want not safe", tt.command)
		}
	}
}

func TestCheckPathTraversal(t *testing.T) {
	tests := []struct {
		command string
		wantOk  bool
	}{
		{"cat file.txt", true},
		{"cat ../file.txt", false},
		{"cat './file.txt'", true},
	}
	for _, tt := range tests {
		result := CheckPathTraversal(tt.command)
		if tt.wantOk && !result.Safe {
			t.Errorf("CheckPathTraversal(%q) = not safe (%s), want safe", tt.command, result.Reason)
		}
		if !tt.wantOk && result.Safe {
			t.Errorf("CheckPathTraversal(%q) = safe, want not safe", tt.command)
		}
	}
}

// --- bashtool.go tests ---

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"1", true},
		{"true", true},
		{"yes", true},
		{"on", true},
		{"TRUE", true},
		{"0", false},
		{"false", false},
		{"", false},
		{"no", false},
	}
	for _, tt := range tests {
		t.Setenv("TEST_TRUTHY_VAR", tt.val)
		got := isTruthy("TEST_TRUTHY_VAR")
		if got != tt.want {
			t.Errorf("isTruthy with value %q = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestBashSecurityEnabled(t *testing.T) {
	// Default: not enabled.
	if BashSecurityEnabled() {
		t.Error("BashSecurityEnabled() should be false by default")
	}

	t.Setenv("CLAUDE_CODE_BASH_SECURITY", "1")
	if !BashSecurityEnabled() {
		t.Error("BashSecurityEnabled() should be true when CLAUDE_CODE_BASH_SECURITY=1")
	}
}
