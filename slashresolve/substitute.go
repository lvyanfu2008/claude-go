package slashresolve

import (
	"regexp"
	"strconv"
	"strings"
)

// ParseArguments splits args using shell-like behavior: simple whitespace split
// when unquoted; aligns with TS parseArguments fallback (split on whitespace).
func ParseArguments(args string) []string {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil
	}
	// Minimal quoted segment support: "a b" and 'c d'
	var out []string
	var b strings.Builder
	inSingle := false
	inDouble := false
	flush := func() {
		s := strings.TrimSpace(b.String())
		b.Reset()
		if s != "" {
			out = append(out, s)
		}
	}
	for _, r := range args {
		switch {
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case (r == ' ' || r == '\t' || r == '\n') && !inSingle && !inDouble:
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	if len(out) == 0 {
		return strings.Fields(args)
	}
	return out
}

// ParseArgumentNames mirrors TS parseArgumentNames: space-separated string or YAML array.
func ParseArgumentNames(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		var out []string
		for _, w := range strings.Fields(s) {
			if isValidArgName(w) {
				out = append(out, w)
			}
		}
		return out
	case []interface{}:
		var out []string
		for _, x := range t {
			if s, ok := x.(string); ok && isValidArgName(strings.TrimSpace(s)) {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

func isValidArgName(s string) bool {
	if s == "" {
		return false
	}
	ok, _ := regexp.MatchString(`^\d+$`, s)
	return !ok
}

func isWordByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// replaceNamedDollar replaces $name when not followed by [ or a word char (TS `(?![\[\w])`).
func replaceNamedDollar(content, name, value string) string {
	needle := "$" + name
	var out strings.Builder
	i := 0
	for i < len(content) {
		j := strings.Index(content[i:], needle)
		if j < 0 {
			out.WriteString(content[i:])
			break
		}
		j += i
		end := j + len(needle)
		if end < len(content) {
			c := content[end]
			if c == '[' || isWordByte(c) {
				out.WriteString(content[i:end])
				i = end
				continue
			}
		}
		out.WriteString(content[i:j])
		out.WriteString(value)
		i = end
	}
	return out.String()
}

func isDigitByte(c byte) bool { return c >= '0' && c <= '9' }

// replaceShorthandDollarDigits replaces $0 $1 … when not followed by a word char (TS `\$(\d+)(?!\w)`).
func replaceShorthandDollarDigits(content string, parsed []string) string {
	var b strings.Builder
	i := 0
	for i < len(content) {
		if content[i] != '$' || i+1 >= len(content) || !isDigitByte(content[i+1]) {
			b.WriteByte(content[i])
			i++
			continue
		}
		j := i + 1
		for j < len(content) && isDigitByte(content[j]) {
			j++
		}
		if j < len(content) && isWordByte(content[j]) {
			b.WriteString(content[i:j])
			i = j
			continue
		}
		idxStr := content[i+1 : j]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			b.WriteString(content[i:j])
			i = j
			continue
		}
		b.WriteString(pick(parsed, idx))
		i = j
	}
	return b.String()
}

// SubstituteArguments mirrors src/utils/argumentSubstitution.ts substituteArguments.
// args is the raw argument string after the command name (may be empty). appendIfNoPlaceholder
// when true appends "\n\nARGUMENTS: {args}" if no substitution changed content and args non-empty.
func SubstituteArguments(content string, args string, appendIfNoPlaceholder bool, argumentNames []string) string {
	original := content
	parsed := ParseArguments(args)

	for i, name := range argumentNames {
		if name == "" {
			continue
		}
		content = replaceNamedDollar(content, name, pick(parsed, i))
	}

	reIdx := regexp.MustCompile(`\$ARGUMENTS\[(\d+)\]`)
	content = reIdx.ReplaceAllStringFunc(content, func(s string) string {
		m := reIdx.FindStringSubmatch(s)
		if len(m) < 2 {
			return s
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			return ""
		}
		return pick(parsed, idx)
	})

	content = replaceShorthandDollarDigits(content, parsed)

	content = strings.ReplaceAll(content, "$ARGUMENTS", args)

	if content == original && appendIfNoPlaceholder && strings.TrimSpace(args) != "" {
		content = content + "\n\nARGUMENTS: " + args
	}
	return content
}

func pick(parsed []string, i int) string {
	if i >= 0 && i < len(parsed) {
		return parsed[i]
	}
	return ""
}
