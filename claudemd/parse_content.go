package claudemd

import (
	"os"
	"path/filepath"
)

const maxIncludeDepth = 5

// ParseMemoryFileContent mirrors claudemd.ts parseMemoryFileContent (pure).
func ParseMemoryFileContent(rawContent, filePath string, typ MemoryType, includeBasePath string) (info *MemoryFileInfo, includePaths []string) {
	if !textExtOK(filePath) {
		return nil, nil
	}
	withoutFrontmatter, globs := ParseFrontmatterPaths(rawContent)
	stripped, _ := StripHTMLCommentsFenceAware(withoutFrontmatter)
	finalContent := stripped
	if typ == MemoryAutoMem || typ == MemoryTeamMem {
		finalContent = TruncateEntrypointContent(stripped)
	}
	contentDiffers := finalContent != rawContent
	if includeBasePath != "" {
		includePaths = ExtractIncludePathsFromMarkdown([]byte(withoutFrontmatter), includeBasePath)
	}
	info = &MemoryFileInfo{
		Path:    filePath,
		Type:    typ,
		Content: finalContent,
		Globs:   append([]string(nil), globs...),
	}
	_ = contentDiffers
	return info, includePaths
}

// ReadMemoryFileFromDisk reads UTF-8 file; missing file → nil, nil.
func ReadMemoryFileFromDisk(filePath string, typ MemoryType, resolvedForInclude string) (*MemoryFileInfo, []string) {
	base := resolvedForInclude
	if base == "" {
		base = filePath
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil
	}
	return ParseMemoryFileContent(string(b), filePath, typ, filepath.Clean(base))
}
