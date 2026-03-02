package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/antopolskiy/kanban-md/internal/task"
)

// PromptFields controls which fields are included in prompt output.
type PromptFields struct {
	Title  bool
	Status bool
	Tags   bool
	Body   bool
}

// DefaultPromptFields returns the default field set for --prompt output.
func DefaultPromptFields() PromptFields {
	return PromptFields{
		Title:  true,
		Status: true,
		Tags:   true,
		Body:   true,
	}
}

// ParsePromptFields parses a comma-separated field list into PromptFields.
// Valid fields: title, status, tags, body. Returns an error for unknown fields.
func ParsePromptFields(spec string) (PromptFields, error) {
	if spec == "" {
		return DefaultPromptFields(), nil
	}

	valid := map[string]bool{
		"title":  true,
		"status": true,
		"tags":   true,
		"body":   true,
	}

	pf := PromptFields{}
	for _, f := range strings.Split(spec, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !valid[f] {
			return pf, fmt.Errorf("unknown prompt field %q (valid: title, status, tags, body)", f)
		}
		switch f {
		case "title":
			pf.Title = true
		case "status":
			pf.Status = true
		case "tags":
			pf.Tags = true
		case "body":
			pf.Body = true
		}
	}

	return pf, nil
}

// TaskPrompt renders a single task in token-efficient prompt format.
func TaskPrompt(w io.Writer, t *task.Task, fields PromptFields) {
	// Header line: Task #ID: Title [status]
	if fields.Title {
		header := "Task #" + strconv.Itoa(t.ID) + ": " + t.Title
		if fields.Status {
			header += " [" + t.Status + "]"
		}
		fmt.Fprintln(w, header)
	} else if fields.Status {
		fmt.Fprintln(w, "Task #"+strconv.Itoa(t.ID)+" ["+t.Status+"]")
	}

	// Tags line.
	if fields.Tags && len(t.Tags) > 0 {
		fmt.Fprintln(w, "Tags: "+strings.Join(t.Tags, ", "))
	}

	// Body: separated by ---.
	if fields.Body && t.Body != "" {
		fmt.Fprintln(w, "---")
		fmt.Fprintln(w, t.Body)
	}
}

// TasksPrompt renders multiple tasks in prompt format with separators.
func TasksPrompt(w io.Writer, tasks []*task.Task, fields PromptFields) {
	for i, t := range tasks {
		if i > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "===")
			fmt.Fprintln(w)
		}
		TaskPrompt(w, t, fields)
	}
}
