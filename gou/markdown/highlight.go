// Package markdown mirrors src/components/Markdown.tsx lexer/cache concepts (marked Token subset).
package markdown

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// HighlightConfig 配置代码高亮
type HighlightConfig struct {
	// Enabled 是否启用代码高亮
	Enabled bool
	// StyleName 高亮样式名称，如 "monokai", "github", "vs" 等
	StyleName string
	// FormatterName 格式化器名称，如 "terminal", "terminal256", "terminal16m"
	FormatterName string
}

// DefaultHighlightConfig 返回默认的高亮配置
func DefaultHighlightConfig() HighlightConfig {
	return HighlightConfig{
		Enabled:       true,
		StyleName:     "monokai",
		FormatterName: "terminal256",
	}
}

// Highlighter 代码高亮器
type Highlighter struct {
	config HighlightConfig
	style  *chroma.Style
	formatter chroma.Formatter
}

// NewHighlighter 创建新的代码高亮器
func NewHighlighter(config HighlightConfig) (*Highlighter, error) {
	h := &Highlighter{
		config: config,
	}

	if !config.Enabled {
		return h, nil
	}

	// 加载样式
	style := styles.Get(config.StyleName)
	if style == nil {
		// 回退到默认样式
		style = styles.Fallback
	}
	h.style = style

	// 创建格式化器
	switch config.FormatterName {
	case "terminal", "terminal8":
		h.formatter = formatters.Get("terminal8")
	case "terminal16":
		h.formatter = formatters.Get("terminal16")
	case "terminal256":
		h.formatter = formatters.Get("terminal256")
	default:
		// 默认使用terminal256
		h.formatter = formatters.Get("terminal256")
	}

	return h, nil
}

// HighlightCode 高亮代码块
func (h *Highlighter) HighlightCode(code, language string) (string, error) {
	if !h.config.Enabled || h.formatter == nil {
		// 高亮未启用或格式化器未初始化，返回原始代码
		return code, nil
	}

	// 确定语言
	var lexer chroma.Lexer
	if language != "" {
		lexer = lexers.Get(language)
	}
	if lexer == nil {
		// 尝试自动检测语言
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		// 使用纯文本
		lexer = lexers.Fallback
	}

	// 获取词法分析器
	lexer = chroma.Coalesce(lexer)

	// 进行词法分析
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, fmt.Errorf("tokenise failed: %w", err)
	}

	// 格式化输出
	var builder strings.Builder
	err = h.formatter.Format(&builder, h.style, iterator)
	if err != nil {
		return code, fmt.Errorf("format failed: %w", err)
	}

	return builder.String(), nil
}

// SupportsLanguage 检查是否支持特定语言
func (h *Highlighter) SupportsLanguage(language string) bool {
	if !h.config.Enabled {
		return false
	}

	lexer := lexers.Get(language)
	return lexer != nil
}

// GetLanguageName 获取语言名称
func (h *Highlighter) GetLanguageName(language string) string {
	if !h.config.Enabled {
		return "plaintext"
	}

	lexer := lexers.Get(language)
	if lexer == nil {
		return "plaintext"
	}

	return lexer.Config().Name
}

// DetectLanguage 检测代码的语言
func (h *Highlighter) DetectLanguage(code string) string {
	if !h.config.Enabled {
		return ""
	}

	lexer := lexers.Analyse(code)
	if lexer == nil {
		return ""
	}

	return lexer.Config().Name
}