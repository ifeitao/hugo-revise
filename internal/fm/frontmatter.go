package fm

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

type Format int

const (
	Unknown Format = iota
	YAML
	TOML
)

type FrontMatter struct {
	Format  Format
	Header  string
	Content string
}

// Parse minimal front matter, supports --- (YAML) and +++ (TOML)
func Parse(input string) (FrontMatter, error) {
	s := strings.TrimSpace(input)
	if strings.HasPrefix(s, "---\n") {
		end := strings.Index(s[4:], "\n---\n")
		if end < 0 {
			return FrontMatter{}, errors.New("yaml front matter not closed")
		}
		header := s[4 : 4+end]
		body := s[4+end+5:]
		return FrontMatter{Format: YAML, Header: header, Content: body}, nil
	}
	if strings.HasPrefix(s, "+++\n") {
		end := strings.Index(s[4:], "\n+++\n")
		if end < 0 {
			return FrontMatter{}, errors.New("toml front matter not closed")
		}
		header := s[4 : 4+end]
		body := s[4+end+5:]
		return FrontMatter{Format: TOML, Header: header, Content: body}, nil
	}
	// No front matter; treat whole as content
	return FrontMatter{Format: Unknown, Header: "", Content: s}, nil
}

// Inject simple key-value into header; minimal and conservative.
// If key is "draft", value is injected as boolean without quotes.
func InjectKV(f FrontMatter, key, value string) (FrontMatter, error) {
	if f.Format == Unknown {
		// default to YAML
		f.Format = YAML
		if key == "draft" {
			f.Header = fmt.Sprintf("%s: %s\n", key, value)
		} else {
			f.Header = fmt.Sprintf("%s: %q\n", key, value)
		}
		return f, nil
	}
	var buf bytes.Buffer
	lines := strings.Split(strings.TrimRight(f.Header, "\n"), "\n")
	replaced := false
	for _, l := range lines {
		if strings.HasPrefix(l, key+":") || strings.HasPrefix(l, key+" =") {
			if f.Format == YAML {
				if key == "draft" {
					buf.WriteString(fmt.Sprintf("%s: %s\n", key, value))
				} else {
					buf.WriteString(fmt.Sprintf("%s: %q\n", key, value))
				}
			} else {
				if key == "draft" {
					buf.WriteString(fmt.Sprintf("%s = %s\n", key, value))
				} else {
					buf.WriteString(fmt.Sprintf("%s = %q\n", key, value))
				}
			}
			replaced = true
		} else {
			buf.WriteString(l + "\n")
		}
	}
	if !replaced {
		if f.Format == YAML {
			if key == "draft" {
				buf.WriteString(fmt.Sprintf("%s: %s\n", key, value))
			} else {
				buf.WriteString(fmt.Sprintf("%s: %q\n", key, value))
			}
		} else {
			if key == "draft" {
				buf.WriteString(fmt.Sprintf("%s = %s\n", key, value))
			} else {
				buf.WriteString(fmt.Sprintf("%s = %q\n", key, value))
			}
		}
	}
	f.Header = buf.String()
	return f, nil
}

// InjectKVUnquoted injects key-value without quotes (for dates, lastmod, etc.)
func InjectKVUnquoted(f FrontMatter, key, value string) (FrontMatter, error) {
	if f.Format == Unknown {
		f.Format = YAML
		f.Header = fmt.Sprintf("%s: %s\n", key, value)
		return f, nil
	}
	var buf bytes.Buffer
	lines := strings.Split(strings.TrimRight(f.Header, "\n"), "\n")
	replaced := false
	for _, l := range lines {
		if strings.HasPrefix(l, key+":") || strings.HasPrefix(l, key+" =") {
			if f.Format == YAML {
				buf.WriteString(fmt.Sprintf("%s: %s\n", key, value))
			} else {
				buf.WriteString(fmt.Sprintf("%s = %s\n", key, value))
			}
			replaced = true
		} else {
			buf.WriteString(l + "\n")
		}
	}
	if !replaced {
		if f.Format == YAML {
			buf.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		} else {
			buf.WriteString(fmt.Sprintf("%s = %s\n", key, value))
		}
	}
	f.Header = buf.String()
	return f, nil
}

// InjectList injects a list/array value for the given key.
// YAML: key:\n  - v1\n  - v2
// TOML: key = ["v1", "v2"]
func InjectList(f FrontMatter, key string, values []string) (FrontMatter, error) {
	if f.Format == Unknown {
		f.Format = YAML
	}

	var rendered string
	if f.Format == YAML {
		var b bytes.Buffer
		b.WriteString(fmt.Sprintf("%s:\n", key))
		for _, v := range values {
			b.WriteString(fmt.Sprintf("  - %s\n", v))
		}
		rendered = b.String()
	} else {
		// TOML array
		var quoted []string
		for _, v := range values {
			quoted = append(quoted, fmt.Sprintf("\"%s\"", v))
		}
		rendered = fmt.Sprintf("%s = [%s]\n", key, strings.Join(quoted, ", "))
	}

	// Replace or append the key
	var buf bytes.Buffer
	lines := strings.Split(strings.TrimRight(f.Header, "\n"), "\n")
	replaced := false
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		// If we find the key line, we need to skip its block (for YAML list) or single line (for TOML)
		if strings.HasPrefix(strings.TrimSpace(l), key+":") || strings.HasPrefix(strings.TrimSpace(l), key+" =") || strings.HasPrefix(strings.TrimSpace(l), key+"=") {
			// For YAML, skip the following indented list items
			if f.Format == YAML {
				// write rendered instead of existing block
				buf.WriteString(rendered)
				replaced = true
				// skip subsequent indented lines (list items)
				for i+1 < len(lines) {
					next := lines[i+1]
					if strings.HasPrefix(next, " ") || strings.HasPrefix(next, "\t") {
						i++
						continue
					}
					break
				}
			} else {
				// TOML: replace this line
				buf.WriteString(rendered)
				replaced = true
			}
		} else {
			buf.WriteString(l + "\n")
		}
	}
	if !replaced {
		buf.WriteString(rendered)
	}

	f.Header = buf.String()
	return f, nil
}

// GetList tries to read a YAML/TOML list into slice of strings.
// Falls back to comma-separated string if present.
func GetList(f FrontMatter, key string) []string {
	var out []string
	lines := strings.Split(strings.TrimRight(f.Header, "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		l := strings.TrimSpace(lines[i])
		if f.Format == YAML {
			if strings.HasPrefix(l, key+":") {
				// collect subsequent indented list items
				for i+1 < len(lines) {
					next := lines[i+1]
					if strings.HasPrefix(next, " ") || strings.HasPrefix(next, "\t") {
						item := strings.TrimSpace(next)
						// expect format "- value"
						item = strings.TrimPrefix(item, "-")
						item = strings.TrimSpace(item)
						out = append(out, item)
						i++
						continue
					}
					break
				}
				break
			}
		} else if f.Format == TOML {
			if strings.HasPrefix(l, key+" =") || strings.HasPrefix(l, key+"=") {
				parts := strings.SplitN(l, "=", 2)
				if len(parts) == 2 {
					arr := strings.TrimSpace(parts[1])
					arr = strings.TrimPrefix(arr, "[")
					arr = strings.TrimSuffix(arr, "]")
					if strings.TrimSpace(arr) != "" {
						items := strings.Split(arr, ",")
						for _, it := range items {
							it = strings.TrimSpace(it)
							it = strings.Trim(it, "\"")
							out = append(out, it)
						}
					}
				}
				break
			}
		}
	}
	if len(out) == 0 {
		// fallback to scalar value (comma-separated)
		s := GetValue(f, key)
		if s != "" {
			parts := strings.Split(s, ",")
			for _, p := range parts {
				out = append(out, strings.TrimSpace(p))
			}
		}
	}
	return out
}

func Stringify(f FrontMatter) string {
	switch f.Format {
	case YAML:
		return fmt.Sprintf("---\n%s\n---\n%s", strings.TrimRight(f.Header, "\n"), f.Content)
	case TOML:
		return fmt.Sprintf("+++\n%s\n+++\n%s", strings.TrimRight(f.Header, "\n"), f.Content)
	default:
		return f.Content
	}
}

// GetValue extracts a field value from front matter (simple string extraction)
func GetValue(f FrontMatter, key string) string {
	lines := strings.Split(f.Header, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if f.Format == YAML {
			if strings.HasPrefix(l, key+":") {
				val := strings.TrimSpace(strings.TrimPrefix(l, key+":"))
				// Remove quotes if present
				val = strings.Trim(val, `"`)
				val = strings.Trim(val, "'")
				return val
			}
		} else if f.Format == TOML {
			if strings.HasPrefix(l, key+" =") || strings.HasPrefix(l, key+"=") {
				parts := strings.SplitN(l, "=", 2)
				if len(parts) == 2 {
					val := strings.TrimSpace(parts[1])
					val = strings.Trim(val, `"`)
					val = strings.Trim(val, "'")
					return val
				}
			}
		}
	}
	return ""
}

// InjectBuildOptions injects build field as proper YAML/TOML structure (Hugo 0.145+)
func InjectBuildOptions(f FrontMatter) (FrontMatter, error) {
	if f.Format == Unknown {
		f.Format = YAML
	}

	var buildBlock string
	if f.Format == YAML {
		buildBlock = "build:\n  list: never\n  render: true\n"
	} else {
		buildBlock = "[build]\nlist = \"never\"\nrender = true\n"
	}

	// Check if build already exists
	if strings.Contains(f.Header, "build:") || strings.Contains(f.Header, "[build]") {
		// Already exists, don't duplicate
		return f, nil
	}

	f.Header = strings.TrimRight(f.Header, "\n") + "\n" + buildBlock
	return f, nil
}

// RemoveKey removes a field from front matter
func RemoveKey(f FrontMatter, key string) (FrontMatter, error) {
	var buf bytes.Buffer
	lines := strings.Split(strings.TrimRight(f.Header, "\n"), "\n")

	for _, l := range lines {
		// Skip lines that start with this key
		if strings.HasPrefix(strings.TrimSpace(l), key+":") ||
			strings.HasPrefix(strings.TrimSpace(l), key+" =") ||
			strings.HasPrefix(strings.TrimSpace(l), key+"=") {
			continue
		}
		buf.WriteString(l + "\n")
	}

	f.Header = buf.String()
	return f, nil
}
