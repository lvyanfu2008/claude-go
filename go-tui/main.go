package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	viewport viewport.Model
	sections []Section // 所有可折叠的区块
	ready    bool
}

// Section 表示一个可折叠的区块
type Section struct {
	Title     string
	Content   string
	Collapsed bool // 该区块是否已折叠
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		}
		// 每次窗口尺寸变化时，都需要基于最新的区块状态重新生成内容
		m.viewport.SetContent(m.buildContent())

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "ctrl+o": // 处理折叠快捷键
			for i := range m.sections {
				m.sections[i].Collapsed = !m.sections[i].Collapsed
			}
			// 状态改变后，重新生成内容并更新 viewport
			m.viewport.SetContent(m.buildContent())
			return m, nil
		}
	}

	// 将剩余的按键（方向键、PageUp等）传递给 viewport 处理滚动
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "正在初始化..."
	}
	return m.viewport.View()
}

// buildContent 根据各区块的折叠状态，动态构建最终显示的内容
func (m *model) buildContent() string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	b.WriteString(titleStyle.Render("📄 滚动查看器（支持折叠）"))
	b.WriteString("\n" + helpStyle.Render("方向键/hjkl 滚动，Ctrl+O 折叠/展开，q 退出"))
	b.WriteString("\n" + strings.Repeat("─", 50))

	// 遍历所有区块，根据折叠状态渲染
	for _, sec := range m.sections {
		b.WriteString("\n\n")
		// 区块标题：显示折叠状态图标
		icon := "▼" // 展开状态
		if sec.Collapsed {
			icon = "▶" // 折叠状态
		}
		header := fmt.Sprintf("%s %s", icon, sec.Title)
		b.WriteString(lipgloss.NewStyle().Bold(true).Render(header))

		if !sec.Collapsed {
			// 展开状态：渲染完整内容
			b.WriteString("\n" + lipgloss.NewStyle().PaddingLeft(2).Render(sec.Content))
		} else {
			// 折叠状态：显示占位符
			b.WriteString("\n" + lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("8")).Render("[内容已折叠...]"))
		}
	}

	return b.String()
}

func main() {
	// 初始化可折叠的区块
	sections := []Section{
		{
			Title:   "使用说明",
			Content: "• 使用 ↑↓ 或 jk 逐行滚动，←→ 或 hl 横向滚动\n• PageUp/PageDown 或 Ctrl+U/Ctrl+D 翻页\n• Home/End 跳至开头/结尾\n• 按 Ctrl+O 可以折叠或展开所有内容区块",
		},
		{
			Title: "第一章：关于 Bubble Tea",
			Content: `Bubble Tea 是一个基于 Elm 架构的 Go 语言 TUI 框架。
它将界面划分为模型（Model）、更新（Update）和视图（View）三部分。
这种清晰的分层让构建复杂的终端应用变得简单而有趣。`,
		},
		{
			Title: "第二章：什么是 Bubbles？",
			Content: `Bubbles 是 Bubble Tea 的官方组件库，提供了许多开箱即用的 TUI 组件。
比如本示例中使用的 viewport，就让我们轻松实现了滚动功能。
此外还有 textinput（文本输入）、table（表格）、list（列表）等实用组件。`,
		},
		{
			Title: "第三章：折叠功能是如何实现的？",
			Content: `这个折叠功能的原理其实很简单：
1. 在模型中维护每个区块的 Collapsed 布尔值。
2. 在 Update 函数中监听 Ctrl+O 按键，并切换这个值。
3. 在渲染内容时，根据 Collapsed 的值决定显示完整内容还是折叠提示。`,
		},
		{
			Title:   "一些长文本",
			Content: strings.Repeat("这是一段非常长的文本，用于展示横向滚动效果。当你使用左右方向键时，可以看到它平滑地移动。", 5),
		},
	}

	m := model{sections: sections}
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Printf("程序运行出错: %v", err)
		os.Exit(1)
	}
}
