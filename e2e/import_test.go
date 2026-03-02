package e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// importOutput mirrors the JSON output of the import command.
type importOutput struct {
	Parent  *int           `json:"parent,omitempty"`
	Created int            `json:"created"`
	Mapping []importResult `json:"mapping"`
}

type importResult struct {
	Ref string `json:"ref"`
	ID  int    `json:"id"`
}

func TestImportBasicJSON(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `{
		"tasks": [
			{"ref": "T0", "title": "First task"},
			{"ref": "T1", "title": "Second task", "depends_on": ["T0"]}
		]
	}`

	specFile := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	var out importOutput
	r := runKanbanJSON(t, kanbanDir, &out, "import", specFile)
	if r.exitCode != 0 {
		t.Fatalf("import failed (exit %d): stdout=%s stderr=%s", r.exitCode, r.stdout, r.stderr)
	}

	if out.Created != 2 {
		t.Errorf("Created = %d, want 2", out.Created)
	}
	if out.Parent != nil {
		t.Errorf("Parent = %v, want nil", out.Parent)
	}
	if len(out.Mapping) != 2 {
		t.Fatalf("Mapping length = %d, want 2", len(out.Mapping))
	}
	if out.Mapping[0].Ref != "T0" || out.Mapping[1].Ref != "T1" {
		t.Errorf("Mapping refs = [%s, %s], want [T0, T1]", out.Mapping[0].Ref, out.Mapping[1].Ref)
	}

	// Verify the second task depends on the first via task file content.
	tasksDir := filepath.Join(kanbanDir, "tasks")
	entries, _ := os.ReadDir(tasksDir)
	for _, e := range entries {
		if strings.Contains(e.Name(), "second") {
			data, _ := os.ReadFile(filepath.Join(tasksDir, e.Name())) //nolint:gosec // test code
			if !strings.Contains(string(data), "depends_on:") {
				t.Errorf("second task file should contain depends_on, got:\n%s", string(data))
			}
		}
	}
}

func TestImportWithParent(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `{
		"parent": {"title": "Epic task", "priority": "high", "tags": ["epic"]},
		"tasks": [
			{"ref": "A", "title": "Subtask A"},
			{"ref": "B", "title": "Subtask B", "depends_on": ["A"]}
		]
	}`

	specFile := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	var out importOutput
	r := runKanbanJSON(t, kanbanDir, &out, "import", specFile)
	if r.exitCode != 0 {
		t.Fatalf("import failed (exit %d): stdout=%s stderr=%s", r.exitCode, r.stdout, r.stderr)
	}

	if out.Parent == nil {
		t.Fatal("Parent should not be nil")
	}
	if *out.Parent != 1 {
		t.Errorf("Parent ID = %d, want 1", *out.Parent)
	}
	if out.Created != 2 {
		t.Errorf("Created = %d, want 2", out.Created)
	}

	// Verify parent task was created with correct fields.
	var parent taskJSON
	runKanbanJSON(t, kanbanDir, &parent, "show", "1")
	if parent.Title != "Epic task" {
		t.Errorf("Parent title = %q, want %q", parent.Title, "Epic task")
	}
	if parent.Priority != priorityHigh {
		t.Errorf("Parent priority = %q, want %q", parent.Priority, priorityHigh)
	}

	// Verify child tasks have parent set.
	tasksDir := filepath.Join(kanbanDir, "tasks")
	entries, _ := os.ReadDir(tasksDir)
	for _, e := range entries {
		if strings.Contains(e.Name(), "subtask") {
			data, _ := os.ReadFile(filepath.Join(tasksDir, e.Name())) //nolint:gosec // test code
			if !strings.Contains(string(data), "parent: 1") {
				t.Errorf("child task %s should have parent: 1, got:\n%s", e.Name(), string(data))
			}
		}
	}
}

func TestImportYAML(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `
tasks:
  - ref: Y0
    title: YAML task one
  - ref: Y1
    title: YAML task two
    depends_on:
      - Y0
`

	specFile := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	var out importOutput
	r := runKanbanJSON(t, kanbanDir, &out, "import", specFile)
	if r.exitCode != 0 {
		t.Fatalf("import YAML failed (exit %d): stdout=%s stderr=%s", r.exitCode, r.stdout, r.stderr)
	}

	if out.Created != 2 {
		t.Errorf("Created = %d, want 2", out.Created)
	}
}

func TestImportStdin(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `{"tasks": [{"ref": "S0", "title": "Stdin task"}]}`

	fullArgs := []string{"--dir", kanbanDir, "--json", "import", "-"}
	cmd := exec.Command(binPath, fullArgs...) //nolint:gosec,noctx // e2e test binary
	cmd.Stdin = strings.NewReader(spec)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("import from stdin failed (exit %d): stderr: %s", exitErr.ExitCode(), stderr.String())
		}
		t.Fatalf("import from stdin failed: %v\nstderr: %s", err, stderr.String())
	}

	var out importOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("parsing output: %v\nstdout: %s", err, stdout.String())
	}

	if out.Created != 1 {
		t.Errorf("Created = %d, want 1", out.Created)
	}
	if out.Mapping[0].Ref != "S0" {
		t.Errorf("Ref = %q, want %q", out.Mapping[0].Ref, "S0")
	}
}

func TestImportEmptyTasksError(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `{"tasks": []}`

	specFile := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	errResp := runKanbanJSONError(t, kanbanDir, "import", specFile)
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want %q", errResp.Code, codeInvalidInput)
	}
}

func TestImportDuplicateRefError(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `{
		"tasks": [
			{"ref": "X", "title": "Task one"},
			{"ref": "X", "title": "Task two"}
		]
	}`

	specFile := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	errResp := runKanbanJSONError(t, kanbanDir, "import", specFile)
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want %q", errResp.Code, codeInvalidInput)
	}
	if !strings.Contains(errResp.Error, "duplicate") {
		t.Errorf("error = %q, should mention 'duplicate'", errResp.Error)
	}
}

func TestImportUnknownDepRefError(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `{
		"tasks": [
			{"ref": "A", "title": "Task A", "depends_on": ["Z"]}
		]
	}`

	specFile := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	errResp := runKanbanJSONError(t, kanbanDir, "import", specFile)
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want %q", errResp.Code, codeInvalidInput)
	}
}

func TestImportSelfRefError(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `{
		"tasks": [
			{"ref": "A", "title": "Task A", "depends_on": ["A"]}
		]
	}`

	specFile := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	errResp := runKanbanJSONError(t, kanbanDir, "import", specFile)
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want %q", errResp.Code, codeInvalidInput)
	}
	if !strings.Contains(errResp.Error, "self-referencing") {
		t.Errorf("error = %q, should mention 'self-referencing'", errResp.Error)
	}
}

func TestImportMissingTitleError(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `{"tasks": [{"ref": "A"}]}`

	specFile := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	errResp := runKanbanJSONError(t, kanbanDir, "import", specFile)
	if errResp.Code != codeInvalidInput {
		t.Errorf("code = %q, want %q", errResp.Code, codeInvalidInput)
	}
}

func TestImportIDSequencing(t *testing.T) {
	kanbanDir := initBoard(t)

	// Create an existing task first to make sure import continues from the right ID.
	mustCreateTask(t, kanbanDir, "Existing task")

	spec := `{
		"parent": {"title": "Parent"},
		"tasks": [
			{"ref": "T0", "title": "Child 1"},
			{"ref": "T1", "title": "Child 2"},
			{"ref": "T2", "title": "Child 3"}
		]
	}`

	specFile := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	var out importOutput
	r := runKanbanJSON(t, kanbanDir, &out, "import", specFile)
	if r.exitCode != 0 {
		t.Fatalf("import failed (exit %d): stdout=%s stderr=%s", r.exitCode, r.stdout, r.stderr)
	}

	// Existing task is #1, parent is #2, children are #3, #4, #5.
	if out.Parent == nil || *out.Parent != 2 {
		t.Errorf("Parent = %v, want 2", out.Parent)
	}
	if out.Mapping[0].ID != 3 || out.Mapping[1].ID != 4 || out.Mapping[2].ID != 5 {
		t.Errorf("IDs = [%d, %d, %d], want [3, 4, 5]",
			out.Mapping[0].ID, out.Mapping[1].ID, out.Mapping[2].ID)
	}

	// Verify next create gets ID 6.
	next := mustCreateTask(t, kanbanDir, "After import")
	if next.ID != 6 {
		t.Errorf("Next task ID = %d, want 6", next.ID)
	}
}

func TestImportHumanOutput(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `{"tasks": [{"ref": "T0", "title": "Human output task"}]}`

	specFile := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	// Run without --json to get human output.
	r := runKanban(t, kanbanDir, "import", specFile)
	if r.exitCode != 0 {
		t.Fatalf("import failed (exit %d): stderr=%s", r.exitCode, r.stderr)
	}

	if !strings.Contains(r.stdout, "Created 1 task") {
		t.Errorf("stdout = %q, want 'Created 1 task'", r.stdout)
	}
	if !strings.Contains(r.stdout, "T0 -> #1") {
		t.Errorf("stdout = %q, want 'T0 -> #1'", r.stdout)
	}
}

func TestImportWithTaskFields(t *testing.T) {
	kanbanDir := initBoard(t)

	spec := `{
		"tasks": [
			{
				"ref": "F0",
				"title": "Full field task",
				"priority": "high",
				"tags": ["backend", "api"],
				"body": "Detailed description",
				"status": "todo",
				"assignee": "alice"
			}
		]
	}`

	specFile := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(specFile, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}

	var out importOutput
	r := runKanbanJSON(t, kanbanDir, &out, "import", specFile)
	if r.exitCode != 0 {
		t.Fatalf("import failed (exit %d): stderr=%s", r.exitCode, r.stderr)
	}

	// Verify the created task has all fields set.
	var tk taskJSON
	runKanbanJSON(t, kanbanDir, &tk, "show", "1")
	if tk.Title != "Full field task" {
		t.Errorf("Title = %q, want %q", tk.Title, "Full field task")
	}
	if tk.Priority != priorityHigh {
		t.Errorf("Priority = %q, want %q", tk.Priority, priorityHigh)
	}
	if tk.Status != statusTodo {
		t.Errorf("Status = %q, want %q", tk.Status, statusTodo)
	}
	if tk.Assignee != assigneeAlice {
		t.Errorf("Assignee = %q, want %q", tk.Assignee, assigneeAlice)
	}
	if strings.TrimSpace(tk.Body) != "Detailed description" {
		t.Errorf("Body = %q, want %q", tk.Body, "Detailed description")
	}
	if len(tk.Tags) != 2 || tk.Tags[0] != "backend" || tk.Tags[1] != "api" {
		t.Errorf("Tags = %v, want [backend api]", tk.Tags)
	}
}
