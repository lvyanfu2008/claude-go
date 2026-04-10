package handwritten

import "goc/utils"

func ptrBool(b bool) *bool          { return &b }
func ptrStr(s string) *string       { return &s }
func ptrInt(n int) *int             { return &n }
func strSlice(s ...string) []string { return append([]string(nil), s...) }

func effortPtr(e utils.EffortValue) *utils.EffortValue {
	v := e
	return &v
}
