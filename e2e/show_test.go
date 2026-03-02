package e2e_test

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Show tests
// ---------------------------------------------------------------------------

func TestShow(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Show me", "--body", "Detailed description",
		"--assignee", assigneeAlice, "--tags", "test")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "show", "1")

	if r.exitCode != 0 {
		t.Fatalf("show failed: %s", r.stderr)
	}
	if task.ID != 1 {
		t.Errorf("ID = %d, want 1", task.ID)
	}
	if task.Title != "Show me" {
		t.Errorf("Title = %q, want %q", task.Title, "Show me")
	}
	if !strings.Contains(task.Body, "Detailed description") {
		t.Errorf("Body = %q, want to contain %q", task.Body, "Detailed description")
	}
	if task.Assignee != assigneeAlice {
		t.Errorf("Assignee = %q, want %q", task.Assignee, assigneeAlice)
	}
}

func TestShowNotFound(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "show", "999")
	if errResp.Code != "TASK_NOT_FOUND" {
		t.Errorf("code = %q, want TASK_NOT_FOUND", errResp.Code)
	}
}

func TestShowInvalidID(t *testing.T) {
	kanbanDir := initBoard(t)

	errResp := runKanbanJSONError(t, kanbanDir, "show", "abc")
	if errResp.Code != "INVALID_TASK_ID" {
		t.Errorf("code = %q, want INVALID_TASK_ID", errResp.Code)
	}
}

// ---------------------------------------------------------------------------
// Edit tests
// ---------------------------------------------------------------------------

func TestShowDisplaysLeadCycleTime(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	runKanban(t, kanbanDir, "--json", "move", "1", statusTodo)
	runKanban(t, kanbanDir, "--json", "move", "1", "done")

	r := runKanban(t, kanbanDir, "--table", "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("show failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Lead time") {
		t.Errorf("show output missing 'Lead time', got: %s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Cycle time") {
		t.Errorf("show output missing 'Cycle time', got: %s", r.stdout)
	}
}

// ---------------------------------------------------------------------------
// Dependency tests
// ---------------------------------------------------------------------------

func TestShowCompactOutput(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Compact show test")

	r := runKanban(t, kanbanDir, "show", "1", "--compact")
	if r.exitCode != 0 {
		t.Fatalf("show --compact failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "Compact show test") {
		t.Error("compact show output should contain task title")
	}
}

// ---------------------------------------------------------------------------
// Prompt output tests
// ---------------------------------------------------------------------------

func TestShowPromptSingleTask(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Prompt test", "--tags", "explorer",
		"--body", "## Summary\nFound chi router pattern.\n\n## Decisions\n1. Use existing middleware")

	r := runKanban(t, kanbanDir, "show", "1", "--prompt")
	if r.exitCode != 0 {
		t.Fatalf("show --prompt failed (exit %d): %s", r.exitCode, r.stderr)
	}
	out := r.stdout

	// Should contain title with status.
	if !strings.Contains(out, "Task #1: Prompt test [backlog]") {
		t.Errorf("missing title line in:\n%s", out)
	}
	// Should contain tags.
	if !strings.Contains(out, "Tags: explorer") {
		t.Errorf("missing tags in:\n%s", out)
	}
	// Should contain body separator and body.
	if !strings.Contains(out, "---") {
		t.Errorf("missing body separator in:\n%s", out)
	}
	if !strings.Contains(out, "## Summary") {
		t.Errorf("missing body content in:\n%s", out)
	}
	// Should NOT contain metadata.
	for _, notWant := range []string{"Created", "Updated", "Assignee", "Estimate"} {
		if strings.Contains(out, notWant) {
			t.Errorf("prompt output should not contain %q:\n%s", notWant, out)
		}
	}
}

func TestShowPromptMultipleTasks(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "First task", "--body", "Body one.")
	mustCreateTask(t, kanbanDir, "Second task", "--tags", "test", "--body", "Body two.")

	r := runKanban(t, kanbanDir, "show", "1,2", "--prompt")
	if r.exitCode != 0 {
		t.Fatalf("show --prompt multi failed (exit %d): %s", r.exitCode, r.stderr)
	}
	out := r.stdout

	if !strings.Contains(out, "Task #1: First task") {
		t.Errorf("missing first task in:\n%s", out)
	}
	if !strings.Contains(out, "===") {
		t.Errorf("missing separator between tasks in:\n%s", out)
	}
	if !strings.Contains(out, "Task #2: Second task") {
		t.Errorf("missing second task in:\n%s", out)
	}
}

func TestShowPromptWithFields(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Fields test", "--tags", "feature", "--body", "Body content.")

	r := runKanban(t, kanbanDir, "show", "1", "--prompt", "--fields", "title,body")
	if r.exitCode != 0 {
		t.Fatalf("show --prompt --fields failed (exit %d): %s", r.exitCode, r.stderr)
	}
	out := r.stdout

	// Title should be present but without status bracket.
	if !strings.Contains(out, "Task #1: Fields test") {
		t.Errorf("missing title in:\n%s", out)
	}
	if strings.Contains(out, "[backlog]") {
		t.Errorf("status should be excluded with --fields title,body:\n%s", out)
	}
	// Tags should NOT be present.
	if strings.Contains(out, "Tags:") {
		t.Errorf("tags should be excluded with --fields title,body:\n%s", out)
	}
	// Body should be present.
	if !strings.Contains(out, "Body content.") {
		t.Errorf("body should be included:\n%s", out)
	}
}

func TestShowPromptInvalidFields(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Invalid fields test")

	r := runKanban(t, kanbanDir, "show", "1", "--prompt", "--fields", "title,invalid")
	if r.exitCode == 0 {
		t.Fatal("expected error for invalid field name")
	}
	if !strings.Contains(r.stderr, "unknown prompt field") {
		t.Errorf("expected field error, got: %s", r.stderr)
	}
}

func TestShowMultipleIDsWithoutPromptFails(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")

	r := runKanban(t, kanbanDir, "show", "1,2")
	if r.exitCode == 0 {
		t.Fatal("expected error for multi-ID without --prompt")
	}
	if !strings.Contains(r.stderr, "multiple IDs only supported with --prompt") {
		t.Errorf("expected multi-ID error, got: %s", r.stderr)
	}
}

func TestShowPromptNoBody(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "No body prompt", "--tags", "quick")

	r := runKanban(t, kanbanDir, "show", "1", "--prompt")
	if r.exitCode != 0 {
		t.Fatalf("show --prompt no body failed (exit %d): %s", r.exitCode, r.stderr)
	}
	out := r.stdout

	if !strings.Contains(out, "Task #1: No body prompt [backlog]") {
		t.Errorf("missing title in:\n%s", out)
	}
	if !strings.Contains(out, "Tags: quick") {
		t.Errorf("missing tags in:\n%s", out)
	}
	// No body means no separator.
	if strings.Contains(out, "---") {
		t.Errorf("should not have body separator when no body:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Children tests
// ---------------------------------------------------------------------------

func TestShowChildrenJSON(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Parent task")
	mustCreateTask(t, kanbanDir, "Child A", "--parent", "1")
	mustCreateTask(t, kanbanDir, "Child B", "--parent", "1")
	mustCreateTask(t, kanbanDir, "Orphan")

	var summary struct {
		Parent struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
		} `json:"parent"`
		Children []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
		} `json:"children"`
		Counts map[string]int `json:"status_counts"`
	}
	r := runKanbanJSON(t, kanbanDir, &summary, "show", "1", "--children")
	if r.exitCode != 0 {
		t.Fatalf("show --children failed: %s", r.stderr)
	}

	if summary.Parent.ID != 1 {
		t.Errorf("parent ID = %d, want 1", summary.Parent.ID)
	}
	if summary.Parent.Title != "Parent task" {
		t.Errorf("parent title = %q, want %q", summary.Parent.Title, "Parent task")
	}
	if len(summary.Children) != 2 {
		t.Fatalf("children count = %d, want 2", len(summary.Children))
	}
	if summary.Counts["backlog"] != 2 {
		t.Errorf("backlog count = %d, want 2", summary.Counts["backlog"])
	}
}

func TestShowChildrenTable(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Parent task")
	mustCreateTask(t, kanbanDir, "Child A", "--parent", "1")
	mustCreateTask(t, kanbanDir, "Child B", "--parent", "1")

	r := runKanban(t, kanbanDir, "--table", "show", "1", "--children")
	if r.exitCode != 0 {
		t.Fatalf("show --children table failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Parent task") {
		t.Errorf("output missing parent title:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Children: 2 tasks") {
		t.Errorf("output missing children count:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Child A") {
		t.Errorf("output missing child A:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Child B") {
		t.Errorf("output missing child B:\n%s", r.stdout)
	}
}

func TestShowChildrenCompact(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Parent task")
	mustCreateTask(t, kanbanDir, "Child A", "--parent", "1")
	mustCreateTask(t, kanbanDir, "Child B", "--parent", "1")

	r := runKanban(t, kanbanDir, "--compact", "show", "1", "--children")
	if r.exitCode != 0 {
		t.Fatalf("show --children compact failed: %s", r.stderr)
	}
	out := r.stdout
	if !strings.Contains(out, "Parent task") {
		t.Errorf("output missing parent title:\n%s", out)
	}
	if !strings.Contains(out, "2 children") {
		t.Errorf("output missing children count:\n%s", out)
	}
	if !strings.Contains(out, "#2 [backlog] Child A") {
		t.Errorf("output missing child A compact line:\n%s", out)
	}
	if !strings.Contains(out, "#3 [backlog] Child B") {
		t.Errorf("output missing child B compact line:\n%s", out)
	}
}

func TestShowChildrenNoChildren(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Lonely task")

	var summary struct {
		Parent struct {
			ID int `json:"id"`
		} `json:"parent"`
		Children []interface{} `json:"children"`
	}
	r := runKanbanJSON(t, kanbanDir, &summary, "show", "1", "--children")
	if r.exitCode != 0 {
		t.Fatalf("show --children with no children failed: %s", r.stderr)
	}
	if summary.Parent.ID != 1 {
		t.Errorf("parent ID = %d, want 1", summary.Parent.ID)
	}
	if len(summary.Children) != 0 {
		t.Errorf("children count = %d, want 0", len(summary.Children))
	}
}

func TestShowChildrenMultipleIDsFails(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Task A")
	mustCreateTask(t, kanbanDir, "Task B")

	r := runKanban(t, kanbanDir, "show", "1,2", "--children")
	if r.exitCode == 0 {
		t.Fatal("expected error for --children with multiple IDs")
	}
	if !strings.Contains(r.stderr, "--children only supports a single task ID") {
		t.Errorf("expected single-ID error, got: %s", r.stderr)
	}
}
