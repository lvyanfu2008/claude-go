package conversation

import "testing"

func TestStoreAddUsage(t *testing.T) {
	var s Store
	s.AddUsage(10, 3)
	s.AddUsage(5, 2)
	if s.UsageInputTotal != 15 || s.UsageOutputTotal != 5 || s.TotalUsageTokens() != 20 {
		t.Fatalf("%+v", s)
	}
}
