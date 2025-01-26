package ohmyglob

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
	"unicode/utf8"
)

var escapeNeededCharRegex = regexp.MustCompile(`[-\/\\^$*+?.()|[\]{}]`)
var runesToEscape []rune

func init() {
	runesToEscape = make([]rune, len(expanders))
}

// Escapes any characters that would have special meaning in a regular expression, returning the escaped string
func escapeRegexComponent(str string) string {
	return escapeNeededCharRegex.ReplaceAllString(str, "\\$0")
}

// separatorsScanner returns a split function for a scanner that returns tokens delimited any of the specified runes.
// Note that the delimiters themselves are counted as tokens, so callers who want to discard the separators must do this
// themselves.
func separatorsScanner(separators []rune) func(data []byte, atEOF bool) (int, []byte, error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// Transform the separators into a map (for efficient lookup)
		seps := make(map[rune]bool)
		for _, r := range separators {
			seps[r] = true
		}

		// Scan until separator, marking the end of a token
		for width, i := 0, 0; i < len(data); i += width {
			var r rune
			r, width = utf8.DecodeRune(data[i:])
			if seps[r] {
				if i == 0 {
					// Separator token
					return i + width, data[0 : i+width], nil
				}

				// Normal token
				return i, data[0:i], nil
			}
		}

		// If we're at EOF, we have a final, non-empty, non-terminated token: return it
		if atEOF && len(data) > 0 {
			return len(data), data[0:], nil
		}

		// Request more data
		return 0, nil, nil
	}
}

// EscapeGlobComponent returns an escaped version of the passed string, ensuring a literal match when used in a pattern.
func EscapeGlobComponent(component string, options *Options) string {
	if options == nil {
		options = DefaultOptions
	}

	runesToEscape := make([]rune, 0, len(expanders)+1)
	runesToEscape = append(runesToEscape, expanders...)
	runesToEscape = append(runesToEscape, options.Separator)

	runesToEscapeMap := make(map[string]bool, len(runesToEscape))
	for _, r := range runesToEscape {
		runesToEscapeMap[string(r)] = true
	}

	scanner := bufio.NewScanner(strings.NewReader(component))
	scanner.Split(separatorsScanner(runesToEscape))
	buf := new(bytes.Buffer)
	for scanner.Scan() {
		component := scanner.Text()
		if runesToEscapeMap[component] {
			buf.WriteRune(Escaper)
		}
		buf.WriteString(component)
	}

	return buf.String()
}

// EscapeGlobString returns an escaped version of the passed string, ensuring a literal match of its components.
// As distinct to EscapeGlobComponent, it will not escape the separator
func EscapeGlobString(gs string, options *Options) string {
	if options == nil {
		options = DefaultOptions
	}

	runesToEscapeMap := make(map[string]bool, len(expanders))
	for _, r := range expanders {
		runesToEscapeMap[string(r)] = true
	}

	scanner := bufio.NewScanner(strings.NewReader(gs))
	scanner.Split(separatorsScanner(expanders))
	buf := new(bytes.Buffer)
	for scanner.Scan() {
		part := scanner.Text()
		if runesToEscapeMap[part] {
			buf.WriteRune(Escaper)
		}
		buf.WriteString(part)
	}

	return buf.String()
}
