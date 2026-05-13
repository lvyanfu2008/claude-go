package app

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ParsedQuestion is a structured question parsed from AskUserQuestion tool input.
type ParsedQuestion struct {
	Question    string         `json:"question"`
	Header      string         `json:"header"`
	Options     []ParsedOption `json:"options"`
	MultiSelect bool           `json:"multiSelect"`
}

// ParsedOption is a single choice within a question.
type ParsedOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Preview     string `json:"preview,omitempty"`
}

// parseAskUserQuestionInput extracts questions from the raw JSON input.
func parseAskUserQuestionInput(input json.RawMessage) ([]ParsedQuestion, error) {
	var in struct {
		Questions []ParsedQuestion `json:"questions"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("AskUserQuestion: invalid input: %w", err)
	}
	if len(in.Questions) < 1 || len(in.Questions) > 4 {
		return nil, fmt.Errorf("AskUserQuestion: questions must have 1-4 entries, got %d", len(in.Questions))
	}
	for i, q := range in.Questions {
		if strings.TrimSpace(q.Question) == "" {
			return nil, fmt.Errorf("AskUserQuestion: question %d has empty text", i+1)
		}
		if len(q.Options) < 2 || len(q.Options) > 4 {
			return nil, fmt.Errorf("AskUserQuestion: question %d needs 2-4 options, got %d", i+1, len(q.Options))
		}
		for j, o := range q.Options {
			if strings.TrimSpace(o.Label) == "" {
				return nil, fmt.Errorf("AskUserQuestion: question %d option %d has empty label", i+1, j+1)
			}
		}
	}
	return in.Questions, nil
}

// questionModel is the Bubble Tea sub-model for interactive question answering.
type questionModel struct {
	questions    []ParsedQuestion
	currentIndex int
	answers      map[string]string   // questionText → comma-joined labels
	annotations  map[string]any      // questionText → {preview, notes}

	cursor      int    // option index (0..len(options)-1) or len(options) for "Chat about this"
	showSubmit  bool   // review screen
	done        bool
	cancelled   bool

	width  int
	height int
	styles questionStyles

	replyCh       chan permissionAskReply
	originalInput json.RawMessage // stored on creation, used to merge answers into updatedInput

	// TODO(Phase 2): textInputMode, textInput for "Other" free-text
}

func newQuestionModel(
	questions []ParsedQuestion,
	replyCh chan permissionAskReply,
	width int,
	height int,
) *questionModel {
	return &questionModel{
		questions:   questions,
		answers:     make(map[string]string),
		annotations: make(map[string]any),
		styles:      newQuestionStyles(width),
		width:       width,
		height:      height,
		replyCh:     replyCh,
	}
}

func (m *questionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *questionModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.showSubmit {
		return m.handleSubmitKey(key)
	}
	return m.handleQuestionKey(key)
}

func (m *questionModel) handleQuestionKey(key string) (tea.Model, tea.Cmd) {
	q := m.questions[m.currentIndex]

	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		maxCursor := len(q.Options) - 1
		if m.cursor < maxCursor {
			m.cursor++
		}
		return m, nil

	case "enter":
		return m.selectCurrentOption()

	case " ":
		if q.MultiSelect {
			return m.toggleMultiSelectOption()
		}
		return m.selectCurrentOption()

	case "tab", "right":
		m.advanceQuestion()
		return m, nil

	case "shift+tab", "left":
		if m.currentIndex > 0 {
			m.currentIndex--
			m.cursor = 0
		}
		return m, nil

	case "esc", "ctrl+c":
		m.cancelled = true
		m.done = true
		return m, nil

	case "y", "Y":
		if m.isLastQuestion() && m.allAnswered() {
			m.showSubmit = true
			return m, nil
		}
	}
	return m, nil
}

func (m *questionModel) handleSubmitKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter", "y", "Y":
		m.done = true
		return m, nil

	case "esc", "n", "N":
		m.showSubmit = false
		return m, nil
	}
	return m, nil
}

func (m *questionModel) selectCurrentOption() (tea.Model, tea.Cmd) {
	q := m.questions[m.currentIndex]
	if m.cursor >= len(q.Options) {
		return m, nil
	}
	opt := q.Options[m.cursor]
	m.answers[q.Question] = opt.Label
	m.advanceQuestion()
	return m, nil
}

func (m *questionModel) toggleMultiSelectOption() (tea.Model, tea.Cmd) {
	q := m.questions[m.currentIndex]
	if m.cursor >= len(q.Options) {
		return m, nil
	}
	opt := q.Options[m.cursor]

	current := m.answers[q.Question]
	var selected []string
	if current != "" {
		selected = strings.Split(current, ", ")
	}

	found := false
	for i, s := range selected {
		if s == opt.Label {
			selected = append(selected[:i], selected[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		selected = append(selected, opt.Label)
	}

	if len(selected) == 0 {
		delete(m.answers, q.Question)
	} else {
		m.answers[q.Question] = strings.Join(selected, ", ")
	}
	return m, nil
}

func (m *questionModel) advanceQuestion() {
	if m.isLastQuestion() {
		m.showSubmit = true
		return
	}
	m.currentIndex++
	m.cursor = 0
}

func (m *questionModel) isLastQuestion() bool {
	return m.currentIndex >= len(m.questions)-1
}

func (m *questionModel) allAnswered() bool {
	for _, q := range m.questions {
		if _, ok := m.answers[q.Question]; !ok {
			return false
		}
	}
	return true
}

func (m *questionModel) IsDone() bool {
	return m.done
}

func (m *questionModel) IsCancelled() bool {
	return m.cancelled
}

// BuildUpdatedInput constructs the tool input with user answers filled in.
// This mirrors the TS submitAnswers() → onAllow(updatedInput) flow.
func (m *questionModel) BuildUpdatedInput(originalInput json.RawMessage) json.RawMessage {
	var in map[string]any
	if err := json.Unmarshal(originalInput, &in); err != nil {
		in = make(map[string]any)
	}
	in["answers"] = m.answers
	if len(m.annotations) > 0 {
		in["annotations"] = m.annotations
	}
	b, _ := json.Marshal(in)
	return b
}

// BuildToolResultText formats the answers for the LLM tool_result.
// Mirrors TS mapToolResultToToolResultBlockParam.
func (m *questionModel) BuildToolResultText() string {
	var parts []string
	for _, q := range m.questions {
		answer, ok := m.answers[q.Question]
		if !ok {
			answer = "(no answer)"
		}
		parts = append(parts, fmt.Sprintf("%q=%q", q.Question, answer))
	}
	return "User has answered your questions: " + strings.Join(parts, ", ")
}

// View renders the question UI.
func (m *questionModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}

	var content string
	if m.showSubmit {
		content = m.renderSubmitView()
	} else {
		content = m.renderQuestionView()
	}
	return tea.NewView(content)
}

func (m *questionModel) renderQuestionView() string {
	q := m.questions[m.currentIndex]

	nav := m.renderNavBar()
	divider := m.styles.divider.Render(strings.Repeat(m.styles.separator, m.width-6))
	title := m.styles.title.Render(q.Question)
	options := m.renderOptions()
	footer := m.renderFooter()

	body := lipgloss.JoinVertical(lipgloss.Left,
		nav,
		divider,
		title,
		"",
		options,
		"",
		footer,
	)

	return m.styles.container.Render(body)
}

func (m *questionModel) renderNavBar() string {
	var tabs []string
	for i, q := range m.questions {
		var tab string
		marker := " "
		if _, ok := m.answers[q.Question]; ok {
			marker = "✓"
		}

		label := q.Header
		if label == "" {
			label = fmt.Sprintf("Q%d", i+1)
		}
		if len(label) > 12 {
			label = label[:11] + "…"
		}

		switch {
		case i == m.currentIndex:
			tab = m.styles.navTabCurrent.Render(fmt.Sprintf("[%s %s]", marker, label))
		case marker == "✓":
			tab = m.styles.navTabAnswered.Render(fmt.Sprintf("[%s %s]", marker, label))
		default:
			tab = m.styles.navTab.Render(fmt.Sprintf("[%s %s]", marker, label))
		}
		tabs = append(tabs, tab)
	}

	submitLabel := "Submit"
	if m.allAnswered() {
		submitLabel = "Submit ✓"
	}
	tabs = append(tabs, m.styles.navTab.Render(submitLabel))

	return m.styles.navBar.Render(strings.Join(tabs, " "))
}

func (m *questionModel) renderOptions() string {
	q := m.questions[m.currentIndex]
	var lines []string

	for i, opt := range q.Options {
		var bullet string
		var style lipgloss.Style

		if q.MultiSelect {
			// Multi-select: show checkbox-style indicators
			selected := m.isOptionSelected(q.Question, opt.Label)
			switch {
			case i == m.cursor:
				if selected {
					bullet = "▶ [✓]"
				} else {
					bullet = "▶ [ ]"
				}
				style = m.styles.optionCursor
			default:
				if selected {
					bullet = "  [✓]"
					style = m.styles.optionSelected
				} else {
					bullet = "  [ ]"
					style = m.styles.option
				}
			}
		} else {
			// Single-select
			switch {
			case i == m.cursor:
				bullet = "●"
				style = m.styles.optionCursor
			case m.answers[q.Question] == opt.Label:
				bullet = "●"
				style = m.styles.optionSelected
			default:
				bullet = "○"
				style = m.styles.option
			}
		}

		line := style.Render(fmt.Sprintf("%s %s", bullet, opt.Label))
		lines = append(lines, line)

		if opt.Description != "" {
			lines = append(lines, m.styles.optionDesc.Render(opt.Description))
		}
	}

	// "Chat about this" option at the bottom
	chatLabel := fmt.Sprintf("%d. Chat about this", len(q.Options)+1)
	if m.cursor == len(q.Options) {
		lines = append(lines, m.styles.optionCursor.Render(chatLabel))
	} else {
		lines = append(lines, m.styles.footer.Render(chatLabel))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *questionModel) isOptionSelected(questionText, label string) bool {
	current := m.answers[questionText]
	if current == "" {
		return false
	}
	for _, s := range strings.Split(current, ", ") {
		if s == label {
			return true
		}
	}
	return false
}

func (m *questionModel) renderFooter() string {
	var parts []string
	if m.questions[m.currentIndex].MultiSelect {
		parts = append(parts, "Space to toggle · Enter to confirm")
	} else {
		parts = append(parts, "Enter to select")
	}
	if len(m.questions) > 1 {
		parts = append(parts, "Tab/←→ to navigate")
	}
	parts = append(parts, "↑↓ to move · Esc to cancel")
	return m.styles.footer.Render(strings.Join(parts, " · "))
}

func (m *questionModel) renderSubmitView() string {
	divider := m.styles.divider.Render(strings.Repeat(m.styles.separator, m.width-6))
	title := m.styles.reviewTitle.Render("Review your answers:")

	var qaLines []string
	for _, q := range m.questions {
		qaLines = append(qaLines, "")
		qaLines = append(qaLines, m.styles.reviewQA.Render(fmt.Sprintf("Q: %s", q.Question)))

		answer, ok := m.answers[q.Question]
		if ok {
			qaLines = append(qaLines, m.styles.reviewAnswer.Render(fmt.Sprintf("→ %s", answer)))
		} else {
			qaLines = append(qaLines, m.styles.reviewWarning.Render("⚠ Not yet answered"))
		}
	}

	footer := lipgloss.JoinVertical(lipgloss.Left,
		"",
		m.styles.footerBold.Render("[Enter/Y] Submit answers   [Esc/N] Go back"),
	)

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		divider,
		lipgloss.JoinVertical(lipgloss.Left, qaLines...),
		footer,
	)

	return m.styles.container.Render(body)
}

func (m *questionModel) Init() tea.Cmd {
	return nil
}
