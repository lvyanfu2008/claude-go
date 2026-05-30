// Builtin status row mirrors TS PromptInputFooter StatusLine + BuiltinStatusLine.tsx
// (model · Context % · tokens · optional cost · Debug mode).

package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"goc/compactservice"
	goccontext "goc/context"
	"goc/modelenv"
)

func gouDemoBuiltinStatusLineDisabled() bool {
	return gouDemoEnvTruthy("GOU_DEMO_NO_BUILTIN_STATUS")
}

func gouDemoDebugModeFooter() bool {
	return gouDemoEnvTruthy("GOU_DEMO_DEBUG") || gouDemoEnvTruthy("CLAUDE_CODE_DEBUG")
}

func shortModelDisplay(modelName string) string {
	parts := strings.Fields(strings.TrimSpace(modelName))
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "gou-demo"
}

func sessionCostUSDFromEnv() float64 {
	v := strings.TrimSpace(os.Getenv("GOU_DEMO_SESSION_COST_USD"))
	if v == "" {
		return 0
	}
	x, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return x
}

func formatCostUSD(x float64) string {
	if x < 0.0001 {
		return "$0"
	}
	return fmt.Sprintf("$%.4f", x)
}

type tokenWarningInfo struct {
	hasData       bool
	percentLeft   int
	isWarning     bool
	isError       bool
	autoCompactOn bool
}

func (m *model) tokenWarningDisplayInfo() tokenWarningInfo {
	if len(m.store.Messages) == 0 {
		return tokenWarningInfo{}
	}
	msgs := compactservice.GetMessagesAfterCompactBoundary(m.store.Messages)
	tokenUsage := compactservice.TokenCountFromLastAPIResponse(msgs)
	if tokenUsage == 0 {
		tokenUsage = compactservice.TokenCountWithEstimation(msgs)
	}
	if tokenUsage == 0 {
		return tokenWarningInfo{}
	}
	model := modelenv.EffectiveMainLoopModel()
	thresholds := compactservice.CompactThresholds{
		ResolveContextWindow:   goccontext.GetContextWindowForModel,
		ResolveMaxOutputTokens: goccontext.GetMaxOutputTokensForModel,
	}
	state := compactservice.CalculateTokenWarningState(tokenUsage, model, nil, thresholds)
	return tokenWarningInfo{
		hasData:       true,
		percentLeft:   state.PercentLeft,
		isWarning:     state.IsAboveWarningThreshold,
		isError:       state.IsAboveErrorThreshold,
		autoCompactOn: compactservice.IsAutoCompactEnabled(),
	}
}

func formatTokenCount(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	}
	return fmt.Sprintf("%d", n)
}


func (m *model) builtinStatusLineView() string {
	if gouDemoBuiltinStatusLineDisabled() || m.uiScreen == gouDemoScreenTranscript {
		return ""
	}
	modelName := modelenv.EffectiveMainLoopModel()
	sep := lipgloss.NewStyle().Faint(true).Render(" │ ")
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Render(shortModelDisplay(modelName)))

	info := m.tokenWarningDisplayInfo()
	if info.hasData {
		b.WriteString(sep)
		pctStr := fmt.Sprintf("Context %d%%", info.percentLeft)
		var ctxStyle lipgloss.Style
		switch {
		case info.isError:
			ctxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		case info.isWarning && !info.autoCompactOn:
			ctxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		default:
			ctxStyle = lipgloss.NewStyle().Faint(true)
		}
		b.WriteString(ctxStyle.Render(pctStr))
	}

	if c := sessionCostUSDFromEnv(); c > 0 {
		b.WriteString(sep)
		b.WriteString(lipgloss.NewStyle().Render(formatCostUSD(c)))
	}
	if gouDemoDebugModeFooter() {
		b.WriteString(sep)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Debug mode"))
	}
	return lipgloss.NewStyle().MaxWidth(m.cols).Render(b.String())
}
