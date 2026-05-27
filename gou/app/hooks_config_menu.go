package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"goc/ccb-engine/diaglog"
	"goc/commands/hooksconfig"
	"goc/gou/pui"
	"goc/hookexec"
	"goc/tools/hookstypes"
)

// hooksMenuMode represents the current level in the 4-level state machine.
type hooksMenuMode int

const (
	hooksModeSelectEvent   hooksMenuMode = 0
	hooksModeSelectMatcher hooksMenuMode = 1
	hooksModeSelectHook    hooksMenuMode = 2
	hooksModeViewHook      hooksMenuMode = 3
)

// hooksConfigMenu is the Bubble Tea sub-model for the /hooks interactive menu.
type hooksConfigMenu struct {
	mode hooksMenuMode

	// Data
	grouped    hooksconfig.GroupedHooks
	eventMetas map[hookstypes.HookEvent]hooksconfig.HookEventMetadata
	toolNames  []string

	// Level 0: SelectEvent
	events      []hookstypes.HookEvent
	eventCursor int

	// Level 1: SelectMatcher
	selectedEvent   hookstypes.HookEvent
	matchers        []string
	matcherCursor   int
	matcherSourceSet map[string]map[hooksconfig.HookSource]bool

	// Level 2: SelectHook
	selectedMatcher string
	hooks           []hooksconfig.IndividualHookConfig
	hookCursor      int

	// Level 3: ViewHook
	selectedHook *hooksconfig.IndividualHookConfig

	// Terminal state
	done bool

	width  int
	height int
	styles hooksMenuStyles
}

// newHooksConfigMenuFromMsg creates a hooksConfigMenu from a GouHooksMenuMsg.
// It loads merged hooks from hookexec and groups them for display.
func (m *model) newHooksConfigMenuFromMsg(msg pui.GouHooksMenuMsg) *hooksConfigMenu {
	cwd := msg.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// Load merged hooks from settings
	table, err := hookexec.MergedHooksForCwd(cwd)
	if err != nil {
		diaglog.Line("[gou/hooks] MergedHooksForCwd error: %v", err)
		table = hookexec.HooksTable{}
	}

	// Convert to IndividualHookConfig list
	var hooks []hooksconfig.IndividualHookConfig
	for evName, matchers := range table {
		ev := hookstypes.HookEvent(evName)
		if !hookstypes.KnownHookEvent(evName) {
			continue
		}
		for _, mg := range matchers {
			for _, rawHook := range mg.Hooks {
				var hc hookstypes.HookCommand
				if err := json.Unmarshal(rawHook, &hc); err != nil {
					continue
				}
				source := hooksconfig.SourceUserSettings
				// Source determination would need tracking from merge paths;
				// default to userSettings for now.
				hooks = append(hooks, hooksconfig.IndividualHookConfig{
					Event:   ev,
					Config:  hc,
					Matcher: mg.Matcher,
					Source:  source,
				})
			}
		}
	}

	// Get tool names for matcher metadata (nil = use empty values for tool-name events)
	var toolNames []string

	// Group by event and matcher
	grouped := hooksconfig.GroupHooksByEventAndMatcher(hooks, toolNames)
	eventMetas := hooksconfig.GetHookEventMetadata(toolNames)

	return newHooksConfigMenu(m.width, m.height, grouped, eventMetas, toolNames)
}

func newHooksConfigMenu(width, height int, grouped hooksconfig.GroupedHooks, eventMetas map[hookstypes.HookEvent]hooksconfig.HookEventMetadata, toolNames []string) *hooksConfigMenu {
	events := make([]hookstypes.HookEvent, 0, len(hookstypes.AllHookEvents))
	for _, ev := range hookstypes.AllHookEvents {
		events = append(events, ev)
	}

	return &hooksConfigMenu{
		mode:        hooksModeSelectEvent,
		grouped:     grouped,
		eventMetas:  eventMetas,
		toolNames:   toolNames,
		events:      events,
		eventCursor: 0,
		width:       width,
		height:      height,
		styles:      newHooksMenuStyles(width),
	}
}

func (m *hooksConfigMenu) Init() tea.Cmd {
	return nil
}

func (m *hooksConfigMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *hooksConfigMenu) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.goBack()
		return m, nil
	case "ctrl+c":
		m.done = true
		return m, nil
	case "up", "k":
		m.moveCursor(-1)
		return m, nil
	case "down", "j":
		m.moveCursor(1)
		return m, nil
	case "enter", " ":
		m.selectCurrent()
		return m, nil
	}
	return m, nil
}

func (m *hooksConfigMenu) goBack() {
	switch m.mode {
	case hooksModeSelectEvent:
		m.done = true
	case hooksModeSelectMatcher:
		m.mode = hooksModeSelectEvent
		m.matcherCursor = 0
	case hooksModeSelectHook:
		if m.eventMetas[m.selectedEvent].MatcherMetadata != nil {
			m.mode = hooksModeSelectMatcher
			m.matcherCursor = 0
		} else {
			m.mode = hooksModeSelectEvent
		}
		m.hookCursor = 0
	case hooksModeViewHook:
		m.mode = hooksModeSelectHook
		m.selectedHook = nil
	}
}

func (m *hooksConfigMenu) moveCursor(delta int) {
	switch m.mode {
	case hooksModeSelectEvent:
		m.eventCursor = clamp(m.eventCursor+delta, 0, len(m.events)-1)
	case hooksModeSelectMatcher:
		m.matcherCursor = clamp(m.matcherCursor+delta, 0, len(m.matchers)-1)
	case hooksModeSelectHook:
		m.hookCursor = clamp(m.hookCursor+delta, 0, len(m.hooks)-1)
	}
}

func (m *hooksConfigMenu) selectCurrent() {
	switch m.mode {
	case hooksModeSelectEvent:
		if m.eventCursor < 0 || m.eventCursor >= len(m.events) {
			return
		}
		ev := m.events[m.eventCursor]
		m.selectedEvent = ev

		if m.eventMetas[ev].MatcherMetadata != nil {
			m.matchers = hooksconfig.SortedMatchersForEvent(m.grouped, ev)
			m.matcherSourceSet = make(map[string]map[hooksconfig.HookSource]bool)
			matcherMap := m.grouped[ev]
			for _, mk := range m.matchers {
				m.matcherSourceSet[mk] = hooksconfig.MatcherSourceSet(matcherMap, mk)
			}
			m.matcherCursor = 0
			m.mode = hooksModeSelectMatcher
		} else {
			m.selectedMatcher = ""
			m.hooks = m.grouped[ev][""]
			m.hookCursor = 0
			m.mode = hooksModeSelectHook
		}

	case hooksModeSelectMatcher:
		if m.matcherCursor < 0 || m.matcherCursor >= len(m.matchers) {
			return
		}
		m.selectedMatcher = m.matchers[m.matcherCursor]
		m.hooks = m.grouped[m.selectedEvent][m.selectedMatcher]
		m.hookCursor = 0
		m.mode = hooksModeSelectHook

	case hooksModeSelectHook:
		if m.hookCursor < 0 || m.hookCursor >= len(m.hooks) {
			return
		}
		hook := m.hooks[m.hookCursor]
		m.selectedHook = &hook
		m.mode = hooksModeViewHook
	}
}

func (m *hooksConfigMenu) IsDone() bool {
	return m.done
}

func (m *hooksConfigMenu) View() tea.View {
	if m.done {
		return tea.NewView("")
	}

	var content string
	switch m.mode {
	case hooksModeSelectEvent:
		content = m.renderSelectEvent()
	case hooksModeSelectMatcher:
		content = m.renderSelectMatcher()
	case hooksModeSelectHook:
		content = m.renderSelectHook()
	case hooksModeViewHook:
		content = m.renderViewHook()
	}

	return tea.NewView(content)
}

// ---- Level 0: SelectEvent ----

func (m *hooksConfigMenu) renderSelectEvent() string {
	divider := m.styles.divider.Render(strings.Repeat(m.styles.separator, m.width-6))

	totalHooks := hooksconfig.TotalHooksCount(m.grouped)
	title := m.styles.title.Render("Hooks")
	subtitle := ""
	if totalHooks == 1 {
		subtitle = m.styles.subtitle.Render("1 hook configured")
	} else if totalHooks > 0 {
		subtitle = m.styles.subtitle.Render(fmt.Sprintf("%d hooks configured", totalHooks))
	} else {
		subtitle = m.styles.subtitle.Render("No hooks configured")
	}

	var lines []string
	for i, ev := range m.events {
		matchers, ok := m.grouped[ev]
		count := 0
		if ok {
			for _, hooks := range matchers {
				count += len(hooks)
			}
		}

		meta := m.eventMetas[ev]
		label := string(ev)
		countStr := ""
		if count > 0 {
			countStr = m.styles.eventCount.Render(fmt.Sprintf(" (%d)", count))
		}

		var line string
		if i == m.eventCursor {
			line = m.styles.eventCursor.Render(fmt.Sprintf("▶ %s%s", label, countStr))
		} else {
			line = m.styles.eventItem.Render(fmt.Sprintf("  %s%s", label, countStr))
		}
		lines = append(lines, line)
		lines = append(lines, m.styles.eventDesc.Render(meta.Summary))
	}

	footer := m.styles.footer.Render("Enter to select · Esc to exit · ↑↓ to move")
	bottomHint := m.styles.footerLink.Render("To add or modify hooks, edit settings.json")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		subtitle,
		divider,
		lipgloss.JoinVertical(lipgloss.Left, lines...),
		"",
		footer,
		bottomHint,
	)

	return m.styles.container.Render(body)
}

// ---- Level 1: SelectMatcher ----

func (m *hooksConfigMenu) renderSelectMatcher() string {
	divider := m.styles.divider.Render(strings.Repeat(m.styles.separator, m.width-6))
	meta := m.eventMetas[m.selectedEvent]

	title := m.styles.title.Render(fmt.Sprintf("%s - Matchers", m.selectedEvent))
	subtitle := m.styles.subtitle.Render(meta.Summary)

	var lines []string
	for i, mk := range m.matchers {
		count := len(m.grouped[m.selectedEvent][mk])

		sources := m.matcherSourceSet[mk]
		sourceStr := formatSourceSet(sources)

		displayMatcher := mk
		if mk == "" {
			displayMatcher = "(all)"
		}

		countStr := ""
		if count > 1 {
			countStr = m.styles.matcherCount.Render(fmt.Sprintf(" (%d hooks)", count))
		} else if count == 1 {
			countStr = m.styles.matcherCount.Render(" (1 hook)")
		}

		var line string
		if i == m.matcherCursor {
			line = m.styles.matcherCursor.Render(fmt.Sprintf("▶ [%s] %s%s", sourceStr, displayMatcher, countStr))
		} else {
			line = m.styles.matcherItem.Render(fmt.Sprintf("  [%s] %s%s", sourceStr, displayMatcher, countStr))
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		lines = append(lines, m.styles.eventDesc.Render("No hooks configured for this event."))
	}

	footer := m.styles.footer.Render("Enter to select · Esc back · ↑↓ to move")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		subtitle,
		divider,
		lipgloss.JoinVertical(lipgloss.Left, lines...),
		"",
		footer,
	)

	return m.styles.container.Render(body)
}

// ---- Level 2: SelectHook ----

func (m *hooksConfigMenu) renderSelectHook() string {
	divider := m.styles.divider.Render(strings.Repeat(m.styles.separator, m.width-6))

	var title string
	if m.eventMetas[m.selectedEvent].MatcherMetadata != nil {
		displayMatcher := m.selectedMatcher
		if displayMatcher == "" {
			displayMatcher = "(all)"
		}
		title = m.styles.title.Render(fmt.Sprintf("%s - Matcher: %s", m.selectedEvent, displayMatcher))
	} else {
		title = m.styles.title.Render(string(m.selectedEvent))
	}

	var lines []string
	for i, hook := range m.hooks {
		typeLabel := hooksconfig.HookTypeLabel(hook.Config)
		typeStyled := m.styles.hookTypeLabel.Render(fmt.Sprintf("[%s]", typeLabel))
		displayText := hooksconfig.HookDisplayText(hook.Config)
		sourceHeader := hooksconfig.SourceHeaderString(hook.Source)

		if len(displayText) > 60 {
			displayText = displayText[:57] + "..."
		}

		var line string
		if i == m.hookCursor {
			line = m.styles.hookCursor.Render(fmt.Sprintf("▶ %s %s", typeStyled, displayText))
		} else {
			line = m.styles.hookItem.Render(fmt.Sprintf("  %s %s", typeStyled, displayText))
		}
		lines = append(lines, line)
		lines = append(lines, m.styles.hookSource.Render(sourceHeader))
	}

	if len(lines) == 0 {
		lines = append(lines, m.styles.eventDesc.Render("No hooks configured for this event."))
		lines = append(lines, m.styles.eventDesc.Render("To add hooks, edit settings.json directly or ask Claude."))
	}

	footer := m.styles.footer.Render("Enter to view details · Esc back · ↑↓ to move")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		divider,
		lipgloss.JoinVertical(lipgloss.Left, lines...),
		"",
		footer,
	)

	return m.styles.container.Render(body)
}

// ---- Level 3: ViewHook ----

func (m *hooksConfigMenu) renderViewHook() string {
	if m.selectedHook == nil {
		return m.styles.container.Render("No hook selected.")
	}

	divider := m.styles.divider.Render(strings.Repeat(m.styles.separator, m.width-6))
	hook := m.selectedHook
	meta := m.eventMetas[hook.Event]

	title := m.styles.title.Render("Hook details")

	// Event
	eventLine := lipgloss.JoinHorizontal(lipgloss.Left,
		m.styles.detailLabel.Render("Event:    "),
		m.styles.detailValue.Render(string(hook.Event)),
	)

	// Matcher
	var matcherLine string
	if meta.MatcherMetadata != nil {
		displayMatcher := hook.Matcher
		if displayMatcher == "" {
			displayMatcher = "(all)"
		}
		matcherLine = lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.detailLabel.Render("Matcher:  "),
			m.styles.detailValue.Render(displayMatcher),
		)
	}

	// Type
	typeLine := lipgloss.JoinHorizontal(lipgloss.Left,
		m.styles.detailLabel.Render("Type:     "),
		m.styles.detailValue.Render(hook.Config.Type),
	)

	// Source
	sourceStr := hooksconfig.SourceDisplayString(hook.Source)
	if hook.PluginName != "" {
		sourceStr += fmt.Sprintf(" (%s)", hook.PluginName)
	}
	sourceLine := lipgloss.JoinHorizontal(lipgloss.Left,
		m.styles.detailLabel.Render("Source:   "),
		m.styles.detailValue.Render(sourceStr),
	)

	// Plugin name (if set)
	var pluginLine string
	if hook.PluginName != "" {
		pluginLine = lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.detailLabel.Render("Plugin:   "),
			m.styles.detailValue.Render(hook.PluginName),
		)
	}

	// Content box
	contentLabel := hooksconfig.HookContentLabel(hook.Config)
	contentValue := hooksconfig.HookContentValue(hook.Config)
	boxLabel := m.styles.detailBoxLabel.Render(contentLabel)
	boxValue := m.styles.detailBoxValue.Render(contentValue)
	contentBox := m.styles.detailBox.Render(lipgloss.JoinVertical(lipgloss.Left, boxLabel, boxValue))

	// Status message
	var statusLine string
	if hook.Config.StatusMessage != "" {
		statusLine = lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.detailLabel.Render("Status:   "),
			m.styles.detailValue.Render(hook.Config.StatusMessage),
		)
	}

	footer := m.styles.footer.Render("Esc to go back")
	bottomHint := m.styles.footerLink.Render("To modify or remove this hook, edit settings.json directly or ask Claude.")

	parts := []string{title, divider, eventLine}
	if matcherLine != "" {
		parts = append(parts, matcherLine)
	}
	parts = append(parts, typeLine, sourceLine)
	if pluginLine != "" {
		parts = append(parts, pluginLine)
	}
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	parts = append(parts, "", contentBox, "", footer, bottomHint)

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return m.styles.container.Render(body)
}

// ---- Helpers ----

func formatSourceSet(sources map[hooksconfig.HookSource]bool) string {
	// Display order: Local > Project > User > Plugin > Session > Built-in
	order := []hooksconfig.HookSource{
		hooksconfig.SourceLocalSettings,
		hooksconfig.SourceProjectSettings,
		hooksconfig.SourceUserSettings,
		hooksconfig.SourcePluginHook,
		hooksconfig.SourceSessionHook,
		hooksconfig.SourceBuiltinHook,
	}
	var parts []string
	for _, src := range order {
		if sources[src] {
			parts = append(parts, hooksconfig.SourceInlineString(src))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
