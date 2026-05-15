package envfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// LoadOptional loads dotenv-style files from dir in order.
// Existing process environment values are preserved; later files override
// values loaded by earlier files.
func LoadOptional(dir string, names ...string) error {
	preserved := map[string]bool{}
	for _, item := range os.Environ() {
		key, _, ok := strings.Cut(item, "=")
		if ok {
			preserved[key] = true
		}
	}
	for _, name := range names {
		path := filepath.Join(dir, name)
		vars, err := parseFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		for key, value := range vars {
			if preserved[key] {
				continue
			}
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("%s: set %s: %w", path, key, err)
			}
		}
	}
	return nil
}

func parseFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	vars := map[string]string{}
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: expected KEY=VALUE", path, lineNo)
		}
		key = strings.TrimSpace(key)
		if !validKey(key) {
			return nil, fmt.Errorf("%s:%d: invalid environment key %q", path, lineNo, key)
		}
		value, err := parseValue(strings.TrimSpace(rawValue))
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		vars[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return vars, nil
}

func validKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func parseValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	switch raw[0] {
	case '"':
		return parseDoubleQuoted(raw)
	case '\'':
		return parseSingleQuoted(raw)
	default:
		return strings.TrimSpace(stripInlineComment(raw)), nil
	}
}

func parseDoubleQuoted(raw string) (string, error) {
	end := -1
	escaped := false
	for i := 1; i < len(raw); i++ {
		switch {
		case escaped:
			escaped = false
		case raw[i] == '\\':
			escaped = true
		case raw[i] == '"':
			end = i
			i = len(raw)
		}
	}
	if end < 0 {
		return "", fmt.Errorf("unterminated double-quoted value")
	}
	tail := strings.TrimSpace(raw[end+1:])
	if tail != "" && !strings.HasPrefix(tail, "#") {
		return "", fmt.Errorf("unexpected content after quoted value")
	}
	return strconv.Unquote(raw[:end+1])
}

func parseSingleQuoted(raw string) (string, error) {
	end := strings.IndexByte(raw[1:], '\'')
	if end < 0 {
		return "", fmt.Errorf("unterminated single-quoted value")
	}
	end++
	tail := strings.TrimSpace(raw[end+1:])
	if tail != "" && !strings.HasPrefix(tail, "#") {
		return "", fmt.Errorf("unexpected content after quoted value")
	}
	return raw[1:end], nil
}

func stripInlineComment(raw string) string {
	for i, r := range raw {
		if r == '#' && (i == 0 || unicode.IsSpace(rune(raw[i-1]))) {
			return raw[:i]
		}
	}
	return raw
}
