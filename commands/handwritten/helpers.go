package handwritten

func ptrBool(b bool) *bool          { return &b }
func ptrStr(s string) *string       { return &s }
func ptrInt(n int) *int             { return &n }
func strSlice(s ...string) []string { return append([]string(nil), s...) }
