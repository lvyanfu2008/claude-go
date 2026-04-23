package localtools

import (
	"strings"
	"unicode"
)

// Curly quotes mirror src/tools/FileEditTool/utils.ts (model emits straight quotes; file may use curly).
const (
	leftSingleCurly  = '\u2018'
	rightSingleCurly = '\u2019'
	leftDoubleCurly  = '\u201c'
	rightDoubleCurly = '\u201d'
)

func normalizeQuotes(str string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(str,
		string(leftSingleCurly), "'"),
		string(rightSingleCurly), "'"),
		string(leftDoubleCurly), `"`),
		string(rightDoubleCurly), `"`)
}

// FindActualString mirrors src/tools/FileEditTool/utils.ts findActualString.
// Quote normalization is 1 rune → 1 rune, so rune indices align between file and normalizeQuotes(file).
func FindActualString(fileContent, searchString string) string {
	if strings.Contains(fileContent, searchString) {
		return searchString
	}
	fr := []rune(fileContent)
	nf := []rune(normalizeQuotes(fileContent))
	ns := []rune(normalizeQuotes(searchString))
	if len(ns) == 0 {
		return ""
	}
	for i := 0; i+len(ns) <= len(nf); i++ {
		ok := true
		for j := range ns {
			if nf[i+j] != ns[j] {
				ok = false
				break
			}
		}
		if ok {
			return string(fr[i : i+len(ns)])
		}
	}
	return ""
}

// PreserveQuoteStyle mirrors src/tools/FileEditTool/utils.ts preserveQuoteStyle (subset when old≠actual).
func PreserveQuoteStyle(oldString, actualOldString, newString string) string {
	if oldString == actualOldString {
		return newString
	}
	hasDouble := strings.ContainsRune(actualOldString, leftDoubleCurly) ||
		strings.ContainsRune(actualOldString, rightDoubleCurly)
	hasSingle := strings.ContainsRune(actualOldString, leftSingleCurly) ||
		strings.ContainsRune(actualOldString, rightSingleCurly)
	out := newString
	if hasDouble {
		out = applyCurlyDoubleQuotes(out)
	}
	if hasSingle {
		out = applyCurlySingleQuotes(out)
	}
	return out
}

func isOpeningContext(chars []rune, index int) bool {
	if index == 0 {
		return true
	}
	prev := chars[index-1]
	return prev == ' ' || prev == '\t' || prev == '\n' || prev == '\r' ||
		prev == '(' || prev == '[' || prev == '{' ||
		prev == '\u2014' || prev == '\u2013'
}

func applyCurlyDoubleQuotes(str string) string {
	chars := []rune(str)
	var b strings.Builder
	for i := 0; i < len(chars); i++ {
		if chars[i] == '"' {
			if isOpeningContext(chars, i) {
				b.WriteRune(leftDoubleCurly)
			} else {
				b.WriteRune(rightDoubleCurly)
			}
		} else {
			b.WriteRune(chars[i])
		}
	}
	return b.String()
}

func applyCurlySingleQuotes(str string) string {
	chars := []rune(str)
	var b strings.Builder
	for i := 0; i < len(chars); i++ {
		if chars[i] == '\'' {
			var prev, next rune
			if i > 0 {
				prev = chars[i-1]
			}
			if i+1 < len(chars) {
				next = chars[i+1]
			}
			prevLetter := unicode.IsLetter(prev)
			nextLetter := unicode.IsLetter(next)
			if prevLetter && nextLetter {
				b.WriteRune(rightSingleCurly)
			} else if isOpeningContext(chars, i) {
				b.WriteRune(leftSingleCurly)
			} else {
				b.WriteRune(rightSingleCurly)
			}
		} else {
			b.WriteRune(chars[i])
		}
	}
	return b.String()
}

// ApplyEditToFile mirrors src/tools/FileEditTool/utils.ts applyEditToFile.
func ApplyEditToFile(originalContent, oldString, newString string, replaceAll bool) string {
	if oldString == "" {
		return newString
	}
	if newString != "" {
		if replaceAll {
			return strings.ReplaceAll(originalContent, oldString, newString)
		}
		return strings.Replace(originalContent, oldString, newString, 1)
	}
	stripTrailingNewline := !strings.HasSuffix(oldString, "\n") &&
		strings.Contains(originalContent, oldString+"\n")
	if stripTrailingNewline {
		if replaceAll {
			return strings.ReplaceAll(originalContent, oldString+"\n", newString)
		}
		return strings.Replace(originalContent, oldString+"\n", newString, 1)
	}
	if replaceAll {
		return strings.ReplaceAll(originalContent, oldString, newString)
	}
	return strings.Replace(originalContent, oldString, newString, 1)
}

// GetPatchForEdit mirrors src/tools/FileEditTool/utils.ts getPatchForEdit.
func GetPatchForEdit(filePath, fileContents, oldString, newString string, replaceAll bool) (patch []StructuredPatchHunk, updatedFile string) {
	updatedFile = ApplyEditToFile(fileContents, oldString, newString, replaceAll)
	if updatedFile == fileContents {
		return nil, updatedFile
	}
	patch = GetPatchFromContents(filePath, fileContents, updatedFile)
	return patch, updatedFile
}
