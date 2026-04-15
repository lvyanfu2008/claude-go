package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Production default is 2s tool-summary delay; tests need stable rebuild counts without timer-driven rebuildHeightCache.
	_ = os.Setenv("GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS", "0")
	os.Exit(m.Run())
}
