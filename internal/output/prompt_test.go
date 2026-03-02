package output

import (
	"strings"
	"testing"
	"time"

	"github.com/antopolskiy/kanban-md/internal/task"
)

func TestTaskPromptDefault(t *testing.T) {
	now := time.Now()
	tk := &task.Task{
		ID: 35, Title: "Explore codebase", Status: "done", Priority: "medium",
		Tags: []string{"explorer"}, Created: now, Updated: now,
		Body: "## Summary\nFound chi router pattern...\n\n## Decisions\n1. Use existing middleware chain",
	}

	var buf strings.Builder
	TaskPrompt(&buf, tk, DefaultPromptFields())
	out := buf.String()

	for _, want := range []string{
		"Task #35: Explore codebase [done]",
		"Tags: explorer",
		"---",
		"## Summary",
		"Found chi router pattern...",
		"## Decisions",
		"1. Use existing middleware chain",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("TaskPrompt missing %q in:\n%s", want, out)
		}
	}

	// Should NOT contain metadata.
	for _, notWant := range []string{
		"Created", "Updated", "Assignee", "Estimate", "Class", "Priority:",
	} {
		if strings.Contains(out, notWant) {
			t.Errorf("TaskPrompt should not contain %q in:\n%s", notWant, out)
		}
	}
}

func TestTaskPromptNoBody(t *testing.T) {
	now := time.Now()
	tk := &task.Task{
		ID: 1, Title: "No body task", Status: "backlog", Priority: "low",
		Created: now, Updated: now,
	}

	var buf strings.Builder
	TaskPrompt(&buf, tk, DefaultPromptFields())
	out := buf.String()

	if strings.Contains(out, "---") {
		t.Errorf("TaskPrompt with no body should not contain separator: %s", out)
	}
	want := "Task #1: No body task [backlog]\n"
	if out != want {
		t.Errorf("TaskPrompt =\n%q\nwant:\n%q", out, want)
	}
}

func TestTaskPromptNoTags(t *testing.T) {
	now := time.Now()
	tk := &task.Task{
		ID: 2, Title: "No tags", Status: "todo", Priority: "medium",
		Created: now, Updated: now, Body: "Some body.",
	}

	var buf strings.Builder
	TaskPrompt(&buf, tk, DefaultPromptFields())
	out := buf.String()

	if strings.Contains(out, "Tags:") {
		t.Errorf("TaskPrompt with no tags should not contain Tags line: %s", out)
	}
}

func TestTaskPromptFieldsSubset(t *testing.T) {
	now := time.Now()
	tk := &task.Task{
		ID: 5, Title: "Selected fields", Status: "in-progress", Priority: "high",
		Tags: []string{"feature"}, Created: now, Updated: now, Body: "Body text.",
	}

	fields := PromptFields{Title: true, Body: true}
	var buf strings.Builder
	TaskPrompt(&buf, tk, fields)
	out := buf.String()

	if !strings.Contains(out, "Task #5: Selected fields") {
		t.Errorf("should contain title: %s", out)
	}
	if strings.Contains(out, "[in-progress]") {
		t.Errorf("should not contain status: %s", out)
	}
	if strings.Contains(out, "Tags:") {
		t.Errorf("should not contain tags: %s", out)
	}
	if !strings.Contains(out, "Body text.") {
		t.Errorf("should contain body: %s", out)
	}
}

func TestTaskPromptStatusOnly(t *testing.T) {
	now := time.Now()
	tk := &task.Task{
		ID: 3, Title: "Status only", Status: "done", Priority: "low",
		Created: now, Updated: now,
	}

	fields := PromptFields{Status: true}
	var buf strings.Builder
	TaskPrompt(&buf, tk, fields)
	out := buf.String()

	want := "Task #3 [done]\n"
	if out != want {
		t.Errorf("TaskPrompt =\n%q\nwant:\n%q", out, want)
	}
}

func TestTasksPromptMultiple(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{ID: 1, Title: "First", Status: "done", Priority: "high", Created: now, Updated: now},
		{
			ID: 2, Title: "Second", Status: "todo", Priority: "medium",
			Tags: []string{"test"}, Created: now, Updated: now, Body: "Body here.",
		},
	}

	var buf strings.Builder
	TasksPrompt(&buf, tasks, DefaultPromptFields())
	out := buf.String()

	if !strings.Contains(out, "Task #1: First [done]") {
		t.Errorf("missing first task: %s", out)
	}
	if !strings.Contains(out, "===") {
		t.Errorf("missing separator between tasks: %s", out)
	}
	if !strings.Contains(out, "Task #2: Second [todo]") {
		t.Errorf("missing second task: %s", out)
	}
	if !strings.Contains(out, "Tags: test") {
		t.Errorf("missing tags on second task: %s", out)
	}
}

func TestTasksPromptSingle(t *testing.T) {
	now := time.Now()
	tasks := []*task.Task{
		{ID: 1, Title: "Only one", Status: "backlog", Priority: "low", Created: now, Updated: now},
	}

	var buf strings.Builder
	TasksPrompt(&buf, tasks, DefaultPromptFields())
	out := buf.String()

	if strings.Contains(out, "===") {
		t.Errorf("single task should not have separator: %s", out)
	}
}

func TestParsePromptFields(t *testing.T) {
	tests := []struct {
		input   string
		want    PromptFields
		wantErr bool
	}{
		{"", DefaultPromptFields(), false},
		{"title", PromptFields{Title: true}, false},
		{"title,status", PromptFields{Title: true, Status: true}, false},
		{"title,status,body,tags", PromptFields{Title: true, Status: true, Body: true, Tags: true}, false},
		{"body", PromptFields{Body: true}, false},
		{"title, body", PromptFields{Title: true, Body: true}, false},
		{"invalid", PromptFields{}, true},
		{"title,invalid", PromptFields{}, true},
	}

	for _, tc := range tests {
		got, err := ParsePromptFields(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParsePromptFields(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("ParsePromptFields(%q) = %+v, want %+v", tc.input, got, tc.want)
		}
	}
}
