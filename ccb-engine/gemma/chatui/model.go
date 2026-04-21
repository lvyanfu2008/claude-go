// Package chatui is a small Bubble Tea terminal chat for [gemma.Client].
// It is standalone tooling under ccb-engine/gemma (not conversation-runtime); multiline input reuses [goc/gou/prompt].
package chatui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goc/ccb-engine/gemma"
	"goc/gou/prompt"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Run starts the TUI until the user quits (Ctrl+C) or the program errors.
func Run() error {
	cfg := ConfigFromEnv()
	cl := gemma.NewClient(cfg)
	m := newModel(cl, cfg)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

type chatResult struct {
	resp *gemma.ChatResponse
	err  error
}

type model struct {
	client *gemma.Client
	cfg    gemma.Config

	vp    viewport.Model
	input prompt.Model
	msgs  []gemma.Message
	busy  bool

	termW int
	termH int
}

func newModel(client *gemma.Client, cfg gemma.Config) *model {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(18))
	vp.SoftWrap = true
	vp.MouseWheelEnabled = true
	vp.FillHeight = true
	in := prompt.New()
	in.SetEnterSubmits(false) // chat: Enter newline, Alt+Enter sends
	return &model{client: client, cfg: cfg, vp: vp, input: in}
}

func (m *model) Init() tea.Cmd {
	return m.vp.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termW, m.termH = msg.Width, msg.Height
		m.applyLayout()
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd

	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd

	case tea.MouseClickMsg:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd

	case chatResult:
		m.busy = false
		if msg.err != nil {
			m.msgs = append(m.msgs, gemma.Message{
				Role:    "assistant",
				Content: "（错误）" + msg.err.Error(),
			})
		} else if msg.resp == nil || len(msg.resp.Choices) == 0 {
			m.msgs = append(m.msgs, gemma.Message{Role: "assistant", Content: "（空回复）"})
		} else {
			m.msgs = append(m.msgs, msg.resp.Choices[0].Message)
		}
		m.refreshLog()
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.busy {
			return m, nil
		}
		_ = m.input.Update(msg)
		if m.input.Submitted() {
			return m.submitComposer()
		}
		m.applyLayout()
		return m, nil

	case tea.PasteMsg:
		if m.busy {
			return m, nil
		}
		_ = m.input.Update(msg)
		m.applyLayout()
		return m, nil
	}

	return m, nil
}

func (m *model) submitComposer() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	m.input.SetValue("")
	if text == "" {
		m.applyLayout()
		return m, nil
	}
	m.msgs = append(m.msgs, gemma.Message{Role: "user", Content: text})
	m.refreshLog()
	m.busy = true
	m.applyLayout()
	return m, m.doChat()
}

func (m *model) doChat() tea.Cmd {
	snapshot := append([]gemma.Message(nil), m.msgs...)
	c := m.client
	cfg := m.cfg
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		resp, err := c.ChatCompletion(ctx, gemma.ChatRequest{
			Model:       cfg.ModelName,
			Messages:    snapshot,
			MaxTokens:   2048,
			Temperature: 0.7,
		})
		return chatResult{resp: resp, err: err}
	}
}

func (m *model) applyLayout() {
	w := m.termW
	if w < 20 {
		w = 80
	}
	h := m.termH
	if h < 10 {
		h = 24
	}
	const headerLines = 4
	minVP := 6
	cl := m.input.LineCount()
	if cl < 3 {
		cl = 3
	}
	maxCL := h - headerLines - minVP - 1
	if maxCL < 3 {
		maxCL = 3
	}
	if cl > maxCL {
		cl = maxCL
	}
	vpH := h - headerLines - cl - 1
	if vpH < minVP {
		vpH = minVP
	}
	m.vp.SetWidth(w)
	m.vp.SetHeight(vpH)
	m.input.SetWidth(w - 2)
}

func (m *model) refreshLog() {
	var b strings.Builder
	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	asstStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sysStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	for _, msg := range m.msgs {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		switch role {
		case "user":
			b.WriteString(userStyle.Render("You"))
		case "system":
			b.WriteString(sysStyle.Render("System"))
		default:
			b.WriteString(asstStyle.Render("Gemma"))
		}
		b.WriteString(": ")
		b.WriteString(msg.Content)
		b.WriteString("\n\n")
	}
	m.vp.SetContent(strings.TrimSuffix(b.String(), "\n"))
	m.vp.GotoBottom()
}

func (m *model) View() tea.View {
	title := lipgloss.NewStyle().Bold(true).Render("gemma-chat — Vertex Gemma (standalone)")
	sub := lipgloss.NewStyle().Faint(true).Render(
		fmt.Sprintf("model=%s  project=%s  Enter=newline  Alt+Enter=send  Ctrl+J=newline  Ctrl+C=quit  wheel=log scroll",
			m.cfg.ModelName, m.cfg.ProjectID),
	)
	if m.busy {
		sub += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("… waiting for model")
	}
	composeHint := lipgloss.NewStyle().Faint(true).Render("Compose:")
	body := strings.Join([]string{title, sub, m.vp.View(), "", composeHint, m.input.View()}, "\n")
	v := tea.NewView(body)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
