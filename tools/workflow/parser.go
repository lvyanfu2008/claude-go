package workflow

import (
	"fmt"
	"strings"
	"unicode"
)

// ParseMeta extracts `export const meta = { ... }` from the script body.
// Returns an error if the meta block contains non-literal expressions
// (variables, function calls, spreads, template interpolation).
func ParseMeta(script string) (*Meta, error) {
	metaStart := findMetaBlockStart(script)
	if metaStart < 0 {
		return nil, fmt.Errorf("workflow: missing 'export const meta = { ... }' block")
	}

	// Find the opening brace after "export const meta ="
	braceStart := strings.Index(script[metaStart:], "{")
	if braceStart < 0 {
		return nil, fmt.Errorf("workflow: expected '{' after 'export const meta ='")
	}
	braceStart += metaStart

	// Find matching closing brace
	braceEnd, err := findBalancedBrace(script, braceStart)
	if err != nil {
		return nil, fmt.Errorf("workflow: meta: %w", err)
	}

	literal := strings.TrimSpace(script[braceStart : braceEnd+1])
	meta, err := parseObjectLiteral(literal)
	if err != nil {
		return nil, fmt.Errorf("workflow: meta: %w", err)
	}

	if meta.Name == "" {
		return nil, fmt.Errorf("workflow: meta.name is required")
	}
	if meta.Description == "" {
		return nil, fmt.Errorf("workflow: meta.description is required")
	}

	return meta, nil
}

// StripExports removes 'export ' prefix from top-level declarations
// so Goja can parse them as globals.
func StripExports(script string) string {
	// Simple replacement: remove "export " from "export const", "export function", etc.
	result := strings.ReplaceAll(script, "export const ", "const ")
	result = strings.ReplaceAll(result, "export function ", "function ")
	result = strings.ReplaceAll(result, "export async ", "async ")
	result = strings.ReplaceAll(result, "export class ", "class ")
	return result
}

// ValidateScript runs basic validation on a workflow script.
func ValidateScript(script string) error {
	_, err := ParseMeta(script)
	return err
}

// findMetaBlockStart finds the start of "export const meta =" in the script.
// Handles optional leading whitespace/comments and variations like:
//
//	export const meta = {
//	export const meta={
//	export const meta  =   {
func findMetaBlockStart(script string) int {
	keywords := []string{
		"export const meta =",
		"export const meta=",
		"export const meta  =",
		"export const meta  =",
	}
	for _, kw := range keywords {
		idx := strings.Index(script, kw)
		if idx >= 0 {
			return idx
		}
	}
	// Case-insensitive fallback
	lower := strings.ToLower(script)
	for _, kw := range keywords {
		idx := strings.Index(lower, kw)
		if idx >= 0 {
			return idx
		}
	}
	return -1
}

// findBalancedBrace finds the matching closing brace for the opening brace at pos.
func findBalancedBrace(s string, openPos int) (int, error) {
	if openPos >= len(s) || s[openPos] != '{' {
		return -1, fmt.Errorf("not an opening brace")
	}
	depth := 0
	inString := false
	inSingleString := false
	inTemplate := false

	for i := openPos; i < len(s); i++ {
		ch := s[i]

		// Handle escape sequences
		if inString || inSingleString || inTemplate {
			if ch == '\\' && i+1 < len(s) {
				i++ // Skip escaped char
				continue
			}
		}

		switch {
		case inString:
			if ch == '"' {
				inString = false
			}
			continue
		case inSingleString:
			if ch == '\'' {
				inSingleString = false
			}
			continue
		case inTemplate:
			if ch == '`' {
				inTemplate = false
			}
			// Detect template interpolation: ${...}
			if ch == '$' && i+1 < len(s) && s[i+1] == '{' {
				return -1, fmt.Errorf("template interpolation is not allowed in meta block")
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '\'':
			inSingleString = true
		case '`':
			inTemplate = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, nil
			}
		case '$':
			// Reject spread operator: ... is handled elsewhere
			// Reject template interpolation: ${ inside strings is caught above
		}
	}
	return -1, fmt.Errorf("unclosed brace")
}

// parseObjectLiteral parses a JSON-like JS object literal string into a Meta struct.
// Only supports string, number, boolean, null, array, and nested object literals.
// Rejects variables, function calls, and template literals.
func parseObjectLiteral(literal string) (*Meta, error) {
	meta := &Meta{}
	p := &parser{s: []rune(literal), pos: 0}

	obj, err := p.parseObject()
	if err != nil {
		return nil, err
	}

	for key, val := range obj {
		switch key {
		case "name":
			s, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("meta.name must be a string")
			}
			meta.Name = s
		case "description":
			s, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("meta.description must be a string")
			}
			meta.Description = s
		case "phases":
			arr, ok := val.([]any)
			if !ok {
				return nil, fmt.Errorf("meta.phases must be an array")
			}
			for i, item := range arr {
				phaseObj, ok := item.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("meta.phases[%d] must be an object", i)
				}
				pm := PhaseMeta{}
				if t, ok := phaseObj["title"].(string); ok {
					pm.Title = t
				}
				if d, ok := phaseObj["detail"].(string); ok {
					pm.Detail = d
				}
				if m, ok := phaseObj["model"].(string); ok {
					pm.Model = m
				}
				meta.Phases = append(meta.Phases, pm)
			}
		}
	}

	return meta, nil
}

// parser is a minimal JS literal parser (JSON-like with JS extensions).
type parser struct {
	s   []rune
	pos int
}

func (p *parser) skipWS() {
	for p.pos < len(p.s) && unicode.IsSpace(p.s[p.pos]) {
		p.pos++
	}
}

func (p *parser) peek() rune {
	if p.pos >= len(p.s) {
		return 0
	}
	return p.s[p.pos]
}

func (p *parser) next() rune {
	ch := p.peek()
	if ch != 0 {
		p.pos++
	}
	return ch
}

func (p *parser) parseObject() (map[string]any, error) {
	p.skipWS()
	if p.next() != '{' {
		return nil, fmt.Errorf("expected '{'")
	}

	result := make(map[string]any)

	p.skipWS()
	if p.peek() == '}' {
		p.next()
		return result, nil
	}

	for {
		p.skipWS()
		// Check for trailing comma
		if p.peek() == '}' {
			p.next()
			return result, nil
		}

		key, err := p.parseKey()
		if err != nil {
			return nil, err
		}

		p.skipWS()
		if p.next() != ':' {
			return nil, fmt.Errorf("expected ':' after key %q", key)
		}

		p.skipWS()
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		result[key] = val

		p.skipWS()
		ch := p.peek()
		if ch == ',' {
			p.next()
			continue
		}
		if ch == '}' {
			p.next()
			return result, nil
		}
		return nil, fmt.Errorf("expected ',' or '}' after value")
	}
}

func (p *parser) parseKey() (string, error) {
	ch := p.peek()
	switch {
	case ch == '\'':
		return p.parseSingleQuotedString()
	case ch == '"':
		return p.parseDoubleQuotedString()
	case ch == '`':
		return "", fmt.Errorf("template literals are not allowed in meta block")
	case unicode.IsLetter(ch) || ch == '_':
		return p.parseIdentifier()
	default:
		return "", fmt.Errorf("unexpected character %q in key position", string(ch))
	}
}

func (p *parser) parseIdentifier() (string, error) {
	start := p.pos
	for p.pos < len(p.s) && (unicode.IsLetter(p.s[p.pos]) || unicode.IsDigit(p.s[p.pos]) || p.s[p.pos] == '_') {
		p.pos++
	}
	return string(p.s[start:p.pos]), nil
}

func (p *parser) parseValue() (any, error) {
	ch := p.peek()
	switch {
	case ch == '{':
		return p.parseObject()
	case ch == '[':
		return p.parseArray()
	case ch == '\'':
		return p.parseSingleQuotedString()
	case ch == '"':
		return p.parseDoubleQuotedString()
	case ch == '`':
		return "", fmt.Errorf("template literals are not allowed in meta block")
	case ch == 't':
		return p.parseLiteralTrue()
	case ch == 'f':
		return p.parseLiteralFalse()
	case ch == 'n':
		return p.parseLiteralNull()
	case unicode.IsDigit(ch) || ch == '-':
		return p.parseNumber()
	default:
		return nil, fmt.Errorf("unexpected character %q in value position", string(ch))
	}
}

func (p *parser) parseArray() ([]any, error) {
	p.next() // consume '['
	var result []any

	p.skipWS()
	if p.peek() == ']' {
		p.next()
		return result, nil
	}

	for {
		p.skipWS()
		// Allow trailing comma: after ',' if next is ']', end array
		if p.peek() == ']' {
			p.next()
			return result, nil
		}
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		result = append(result, val)

		p.skipWS()
		ch := p.next()
		if ch == ',' {
			continue
		}
		if ch == ']' {
			return result, nil
		}
		return nil, fmt.Errorf("expected ',' or ']' in array")
	}
}

func (p *parser) parseDoubleQuotedString() (string, error) {
	p.next() // consume opening "
	var result []rune
	for p.pos < len(p.s) {
		ch := p.next()
		if ch == '\\' && p.pos < len(p.s) {
			nextCh := p.next()
			switch nextCh {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case '\\', '"', '\'':
				result = append(result, nextCh)
			default:
				result = append(result, '\\', nextCh)
			}
			continue
		}
		if ch == '"' {
			return string(result), nil
		}
		result = append(result, ch)
	}
	return "", fmt.Errorf("unterminated double-quoted string")
}

func (p *parser) parseSingleQuotedString() (string, error) {
	p.next() // consume opening '
	var result []rune
	for p.pos < len(p.s) {
		ch := p.next()
		if ch == '\\' && p.pos < len(p.s) {
			nextCh := p.next()
			switch nextCh {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case '\\', '\'', '"':
				result = append(result, nextCh)
			default:
				result = append(result, '\\', nextCh)
			}
			continue
		}
		if ch == '\'' {
			return string(result), nil
		}
		result = append(result, ch)
	}
	return "", fmt.Errorf("unterminated single-quoted string")
}

func (p *parser) parseLiteralTrue() (any, error) {
	if p.pos+4 <= len(p.s) && string(p.s[p.pos:p.pos+4]) == "true" {
		p.pos += 4
		return true, nil
	}
	return nil, fmt.Errorf("expected 'true'")
}

func (p *parser) parseLiteralFalse() (any, error) {
	if p.pos+5 <= len(p.s) && string(p.s[p.pos:p.pos+5]) == "false" {
		p.pos += 5
		return false, nil
	}
	return nil, fmt.Errorf("expected 'false'")
}

func (p *parser) parseLiteralNull() (any, error) {
	if p.pos+4 <= len(p.s) && string(p.s[p.pos:p.pos+4]) == "null" {
		p.pos += 4
		return nil, nil
	}
	return nil, fmt.Errorf("expected 'null'")
}

func (p *parser) parseNumber() (any, error) {
	start := p.pos
	if p.peek() == '-' {
		p.pos++
	}
	for p.pos < len(p.s) && unicode.IsDigit(p.s[p.pos]) {
		p.pos++
	}
	if p.pos < len(p.s) && p.s[p.pos] == '.' {
		p.pos++
		for p.pos < len(p.s) && unicode.IsDigit(p.s[p.pos]) {
			p.pos++
		}
	}
	if start == p.pos {
		return nil, fmt.Errorf("expected number")
	}
	// Return the number as a float64 for simplicity.
	numStr := string(p.s[start:p.pos])
	var f float64
	fmt.Sscanf(numStr, "%f", &f)
	return f, nil
}
