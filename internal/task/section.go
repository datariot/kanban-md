package task

import (
	"strings"
)

// Section represents a named section within a task body.
type Section struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

// ParseSections splits a task body into sections delimited by ## headings.
// Content before the first ## heading is returned as a section with an empty name.
func ParseSections(body string) []Section {
	if body == "" {
		return nil
	}

	var sections []Section
	lines := strings.Split(body, "\n")

	var currentName string
	var currentLines []string

	for _, line := range lines {
		if name, ok := parseSectionHeading(line); ok {
			// Flush the previous section.
			sections = appendSection(sections, currentName, currentLines)
			currentName = name
			currentLines = nil
		} else {
			currentLines = append(currentLines, line)
		}
	}

	// Flush the last section.
	sections = appendSection(sections, currentName, currentLines)
	return sections
}

// GetSection extracts a single named section from the body.
// Returns the section body and true if found, or empty string and false.
func GetSection(body, name string) (string, bool) {
	for _, s := range ParseSections(body) {
		if strings.EqualFold(s.Name, name) {
			return s.Body, true
		}
	}
	return "", false
}

// SetSection creates or replaces a named section in the body.
// If the section already exists, its content is replaced.
// If not, the section is appended at the end.
func SetSection(body, name, content string) string {
	sections := ParseSections(body)

	found := false
	for i, s := range sections {
		if strings.EqualFold(s.Name, name) {
			sections[i].Body = content
			found = true
			break
		}
	}

	if !found {
		sections = append(sections, Section{Name: name, Body: content})
	}

	return renderSections(sections)
}

// parseSectionHeading checks if a line is a ## heading and returns the name.
func parseSectionHeading(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "## ") {
		name := strings.TrimSpace(trimmed[3:])
		if name != "" {
			return name, true
		}
	}
	return "", false
}

// appendSection adds a section to the list, trimming trailing blank lines from the body.
func appendSection(sections []Section, name string, lines []string) []Section {
	body := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	// Also trim leading newline that separates heading from content.
	body = strings.TrimLeft(body, "\n")

	// Skip completely empty unnamed preamble sections.
	if name == "" && body == "" {
		return sections
	}

	return append(sections, Section{Name: name, Body: body})
}

// renderSections rebuilds the body from sections.
func renderSections(sections []Section) string {
	var b strings.Builder

	for i, s := range sections {
		if i > 0 {
			b.WriteString("\n\n")
		}

		if s.Name != "" {
			b.WriteString("## ")
			b.WriteString(s.Name)
			b.WriteString("\n")
		}

		if s.Body != "" {
			b.WriteString(s.Body)
		}
	}

	return b.String()
}
