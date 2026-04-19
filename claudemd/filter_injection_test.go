package claudemd

import (
	"os"
	"strings"
	"testing"
)

func TestMemoryFileFilteringAndInjection(t *testing.T) {
	// 测试1：MemoryFileInfo 结构体是否包含必要字段
	t.Run("MemoryFileInfo structure", func(t *testing.T) {
		// 检查 MemoryFileInfo 结构体定义
		info := MemoryFileInfo{
			Path:    "/test/path.md",
			Type:    MemoryProject,
			Content: "test content",
			Globs:   []string{"*.go"},
		}
		
		if info.Path != "/test/path.md" {
			t.Errorf("Path field missing or incorrect")
		}
		if info.Type != MemoryProject {
			t.Errorf("Type field missing or incorrect")
		}
		if info.Content != "test content" {
			t.Errorf("Content field missing or incorrect")
		}
		if len(info.Globs) != 1 || info.Globs[0] != "*.go" {
			t.Errorf("Globs field missing or incorrect")
		}
		
		// 检查是否缺少 RawContent 和 ContentDiffersFromDisk 字段
		// 这些字段在当前实现中不存在
	})
	
	// 测试2：FilterInjectedMemoryFiles 基本功能
	t.Run("FilterInjectedMemoryFiles basic", func(t *testing.T) {
		files := []MemoryFileInfo{
			{Type: MemoryProject, Content: "project"},
			{Type: MemoryAutoMem, Content: "auto-memory"},
			{Type: MemoryTeamMem, Content: "team-memory"},
			{Type: MemoryLocal, Content: "local"},
		}
		
		// 测试环境变量未设置时，不过滤
		os.Unsetenv("CLAUDE_CODE_TENGU_MOTH_COPSE")
		filtered := FilterInjectedMemoryFiles(files)
		if len(filtered) != 4 {
			t.Errorf("expected 4 files when flag not set, got %d", len(filtered))
		}
		
		// 测试环境变量设置时，过滤 auto-memory 和 team-memory
		os.Setenv("CLAUDE_CODE_TENGU_MOTH_COPSE", "1")
		filtered = FilterInjectedMemoryFiles(files)
		if len(filtered) != 2 {
			t.Errorf("expected 2 files when flag set, got %d", len(filtered))
		}
		
		// 检查过滤后的文件类型
		hasProject := false
		hasLocal := false
		for _, f := range filtered {
			if f.Type == MemoryProject {
				hasProject = true
			}
			if f.Type == MemoryLocal {
				hasLocal = true
			}
		}
		if !hasProject || !hasLocal {
			t.Errorf("filtered files missing expected types")
		}
		
		os.Unsetenv("CLAUDE_CODE_TENGU_MOTH_COPSE")
	})
	
	// 测试3：ParseMemoryFileContent 内容差异检测
	t.Run("ParseMemoryFileContent content differs", func(t *testing.T) {
		rawContent := "# Original\nSome content <!-- comment --> more content"
		filePath := "/test.md"
		
		info, _ := ParseMemoryFileContent(rawContent, filePath, MemoryProject, "")
		if info == nil {
			t.Fatal("ParseMemoryFileContent returned nil")
		}
		
		// 检查内容是否被处理（HTML 注释被移除）
		if strings.Contains(info.Content, "<!-- comment -->") {
			t.Error("HTML comment not stripped from content")
		}
		
		// 注意：当前实现中 contentDiffers 被计算但没有存储
		// 我们需要扩展 MemoryFileInfo 来存储这个信息
	})
	
	// 测试4：自动记忆和团队记忆的内容截断
	t.Run("AutoMem and TeamMem truncation", func(t *testing.T) {
		longContent := strings.Repeat("This is a very long content. ", 1000) // 约 30,000 字符
		
		// 测试普通记忆类型
		info, _ := ParseMemoryFileContent(longContent, "/test.md", MemoryProject, "")
		if info == nil {
			t.Fatal("ParseMemoryFileContent returned nil")
		}
		
		// 普通记忆类型不应该被截断（除非超过总限制）
		if len(info.Content) < len(longContent) {
			// 注意：可能因为 HTML 注释剥离而变短，但不是截断
		}
		
		// 测试自动记忆类型
		info2, _ := ParseMemoryFileContent(longContent, "/test.md", MemoryAutoMem, "")
		if info2 == nil {
			t.Fatal("ParseMemoryFileContent returned nil for AutoMem")
		}
		
		// 自动记忆应该被 TruncateEntrypointContent 截断
		// 我们需要检查是否调用了 TruncateEntrypointContent
	})
	
	// 测试5：原始内容缓存需求
	t.Run("Raw content caching need", func(t *testing.T) {
		// 当前实现中，ParseMemoryFileContent 接收 rawContent 参数
		// 但返回的 MemoryFileInfo 只包含处理后的 Content
		// 没有存储原始内容
		
		rawContent := "# Original\n<!-- comment -->Content"
		info, _ := ParseMemoryFileContent(rawContent, "/test.md", MemoryProject, "")
		
		// 当前实现中，我们无法获取原始内容
		// 需要添加 RawContent 字段到 MemoryFileInfo
		_ = info // 使用变量避免编译错误
	})
}