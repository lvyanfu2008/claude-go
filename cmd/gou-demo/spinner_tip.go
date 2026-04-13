package main

import (
	"os"
	"strings"
	"time"
)

// TS Spinner.tsx effectiveTip priority (simplified: no nextTask / contextTipsActive / spinnerTip from AppState).

const (
	spinnerTipClear       = "Use /clear to start fresh when switching topics and free up context"
	spinnerTipBtw        = "Use /btw to ask a quick side question without interrupting Claude's current work"
	spinnerTipPromptQueue = "Hit Enter to queue up additional messages while Claude is working."
)

func gouDemoSpinnerTipsEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("CLAUDE_CODE_SPINNER_TIPS_ENABLED")))
	if v == "0" || v == "false" || v == "off" || v == "no" {
		return false
	}
	v2 := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_SPINNER_TIPS")))
	if v2 == "0" || v2 == "false" || v2 == "off" || v2 == "no" {
		return false
	}
	return true
}

func effectiveSpinnerTip(elapsed time.Duration, tipsEnabled bool) string {
	if !tipsEnabled {
		return ""
	}
	if elapsed >= 30*time.Minute {
		return spinnerTipClear
	}
	if elapsed >= 30*time.Second {
		return spinnerTipBtw
	}
	return spinnerTipPromptQueue
}

// gouSpinnerTickMsg drives spinner punctuation animation while queryBusy (TS SpinnerAnimationRow ~50ms).
type gouSpinnerTickMsg struct{}
