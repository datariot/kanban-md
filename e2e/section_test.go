package e2e_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Section tests
// ---------------------------------------------------------------------------

func TestEditSetSection_CreateNew(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Section target", "--body", "Some preamble text")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1",
		"--set-section", "Artifact", "--section-body", "Result: success")
	if r.exitCode != 0 {
		t.Fatalf("edit --set-section failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(task.Body, "## Artifact") {
		t.Errorf("body should contain '## Artifact', got %q", task.Body)
	}
	if !strings.Contains(task.Body, "Result: success") {
		t.Errorf("body should contain section content, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "Some preamble text") {
		t.Errorf("body should preserve preamble, got %q", task.Body)
	}
}

func TestEditSetSection_ReplaceExisting(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Replace target",
		"--body", "## Artifact\nOld content\n\n## Decisions\n1. Use chi")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1",
		"--set-section", "Artifact", "--section-body", "New content")
	if r.exitCode != 0 {
		t.Fatalf("edit --set-section failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(task.Body, "New content") {
		t.Errorf("body should contain updated content, got %q", task.Body)
	}
	if strings.Contains(task.Body, "Old content") {
		t.Errorf("body should not contain old content, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "## Decisions") {
		t.Errorf("body should preserve other sections, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "1. Use chi") {
		t.Errorf("body should preserve other section content, got %q", task.Body)
	}
}

func TestEditSetSection_EmptyBody(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Empty body section")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "edit", "1",
		"--set-section", "Notes", "--section-body", "First note")
	if r.exitCode != 0 {
		t.Fatalf("edit --set-section failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(task.Body, "## Notes") {
		t.Errorf("body should contain section heading, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "First note") {
		t.Errorf("body should contain section content, got %q", task.Body)
	}
}

func TestEditSetSection_ConflictsWithBody(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Conflict target")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1",
		"--body", "replace", "--set-section", "Notes", "--section-body", "content")
	if errResp.Code != codeStatusConflict {
		t.Errorf("code = %q, want STATUS_CONFLICT", errResp.Code)
	}
}

func TestEditSetSection_ConflictsWithAppendBody(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Conflict target")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1",
		"--append-body", "append", "--set-section", "Notes", "--section-body", "content")
	if errResp.Code != codeStatusConflict {
		t.Errorf("code = %q, want STATUS_CONFLICT", errResp.Code)
	}
}

func TestEditSetSection_EmptyName(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Empty section name")

	errResp := runKanbanJSONError(t, kanbanDir, "edit", "1",
		"--set-section", "", "--section-body", "content")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
}

func TestShowSection(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Show section target",
		"--body", "## Artifact\nResult: success\nSummary: done\n\n## Decisions\n1. Use chi")

	r := runKanban(t, kanbanDir, "show", "1", "--section", "Artifact")
	if r.exitCode != 0 {
		t.Fatalf("show --section failed (exit %d): %s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "Result: success") {
		t.Errorf("output should contain section content, got %q", r.stdout)
	}
	// Should NOT contain the other section.
	if strings.Contains(r.stdout, "Decisions") {
		t.Errorf("output should only contain the requested section, got %q", r.stdout)
	}
}

func TestShowSection_JSON(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "JSON section target",
		"--body", "## Artifact\nResult: success\n\n## Notes\nSome notes")

	r := runKanban(t, kanbanDir, "--json", "show", "1", "--section", "Artifact")
	if r.exitCode != 0 {
		t.Fatalf("show --section --json failed (exit %d): %s", r.exitCode, r.stderr)
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(r.stdout), &result); err != nil {
		t.Fatalf("parsing JSON: %v\nstdout: %s", err, r.stdout)
	}
	if result["section"] != "Artifact" {
		t.Errorf("section = %q, want Artifact", result["section"])
	}
	if result["content"] != "Result: success" {
		t.Errorf("content = %q, want 'Result: success'", result["content"])
	}
}

func TestShowSection_NotFound(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Missing section", "--body", "No sections here")

	errResp := runKanbanJSONError(t, kanbanDir, "show", "1", "--section", "Missing")
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want INVALID_INPUT", errResp.Code)
	}
}

func TestEditSetSection_MultipleSections(t *testing.T) {
	kanbanDir := initBoard(t)
	mustCreateTask(t, kanbanDir, "Multi section target")

	// Add first section.
	runKanban(t, kanbanDir, "--json", "edit", "1",
		"--set-section", "Artifact", "--section-body", "Result: success")
	// Add second section.
	runKanban(t, kanbanDir, "--json", "edit", "1",
		"--set-section", "Decisions", "--section-body", "1. Use chi router")

	var task taskJSON
	r := runKanbanJSON(t, kanbanDir, &task, "show", "1")
	if r.exitCode != 0 {
		t.Fatalf("show failed: %s", r.stderr)
	}
	if !strings.Contains(task.Body, "## Artifact") {
		t.Errorf("body should contain Artifact section, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "## Decisions") {
		t.Errorf("body should contain Decisions section, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "Result: success") {
		t.Errorf("body should contain Artifact content, got %q", task.Body)
	}
	if !strings.Contains(task.Body, "1. Use chi router") {
		t.Errorf("body should contain Decisions content, got %q", task.Body)
	}
}
