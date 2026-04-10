package claudemd

import "strings"

// TextFileExtensionsAllowedForInclude mirrors claudemd.ts TEXT_FILE_EXTENSIONS.
var TextFileExtensionsAllowedForInclude = map[string]struct{}{
	".md": {}, ".txt": {}, ".text": {}, ".json": {}, ".yaml": {}, ".yml": {}, ".toml": {},
	".xml": {}, ".csv": {}, ".html": {}, ".htm": {}, ".css": {}, ".scss": {}, ".sass": {},
	".less": {}, ".js": {}, ".ts": {}, ".tsx": {}, ".jsx": {}, ".mjs": {}, ".cjs": {},
	".mts": {}, ".cts": {}, ".py": {}, ".pyi": {}, ".pyw": {}, ".rb": {}, ".erb": {},
	".rake": {}, ".go": {}, ".rs": {}, ".java": {}, ".kt": {}, ".kts": {}, ".scala": {},
	".c": {}, ".cpp": {}, ".cc": {}, ".cxx": {}, ".h": {}, ".hpp": {}, ".hxx": {},
	".cs": {}, ".swift": {}, ".sh": {}, ".bash": {}, ".zsh": {}, ".fish": {}, ".ps1": {},
	".bat": {}, ".cmd": {}, ".env": {}, ".ini": {}, ".cfg": {}, ".conf": {}, ".config": {},
	".properties": {}, ".sql": {}, ".graphql": {}, ".gql": {}, ".proto": {}, ".vue": {},
	".svelte": {}, ".astro": {}, ".ejs": {}, ".hbs": {}, ".pug": {}, ".jade": {},
	".php": {}, ".pl": {}, ".pm": {}, ".lua": {}, ".r": {}, ".R": {}, ".dart": {},
	".ex": {}, ".exs": {}, ".erl": {}, ".hrl": {}, ".clj": {}, ".cljs": {}, ".cljc": {},
	".edn": {}, ".hs": {}, ".lhs": {}, ".elm": {}, ".ml": {}, ".mli": {}, ".f": {},
	".f90": {}, ".f95": {}, ".for": {}, ".cmake": {}, ".make": {}, ".makefile": {},
	".gradle": {}, ".sbt": {}, ".rst": {}, ".adoc": {}, ".asciidoc": {}, ".org": {},
	".tex": {}, ".latex": {}, ".lock": {}, ".log": {}, ".diff": {}, ".patch": {},
}

func textExtOK(path string) bool {
	ext := ""
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			ext = path[i:]
			break
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	if ext == "" {
		return true
	}
	_, ok := TextFileExtensionsAllowedForInclude[strings.ToLower(ext)]
	return ok
}
