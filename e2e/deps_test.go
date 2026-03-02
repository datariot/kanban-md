package e2e_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// deps command tests
// ---------------------------------------------------------------------------

// setupDepsBoard creates a board with a dependency graph:
//
//	#1 "Base task" (done)
//	#2 "Middle task" depends_on [1] (in-progress)
//	#3 "Leaf task" depends_on [2] (backlog)
//	#4 "Isolated task" (backlog)
func setupDepsBoard(t *testing.T) string {
	t.Helper()
	dir := initBoard(t)
	mustCreateTask(t, dir, "Base task")
	mustCreateTask(t, dir, "Middle task", "--depends-on", "1")
	mustCreateTask(t, dir, "Leaf task", "--depends-on", "2")
	mustCreateTask(t, dir, "Isolated task")
	// Move task 1 to done so it's a resolved dep.
	runKanban(t, dir, "move", "1", "done")
	// Claim and move task 2 to in-progress (require_claim enforced).
	runKanban(t, dir, "edit", "2", "--claim", "test-agent")
	runKanban(t, dir, "move", "2", "in-progress", "--claim", "test-agent")
	return dir
}

func TestDepsShowBothDirections(t *testing.T) {
	dir := setupDepsBoard(t)

	// Task 2 depends on 1 (upstream) and 3 depends on 2 (downstream).
	var result struct {
		TaskID     int                `json:"task_id"`
		Upstream   []struct{ ID int } `json:"upstream"`
		Downstream []struct{ ID int } `json:"downstream"`
	}
	r := runKanbanJSON(t, dir, &result, "deps", "2")
	if r.exitCode != 0 {
		t.Fatalf("deps failed: %s", r.stderr)
	}
	if result.TaskID != 2 {
		t.Errorf("task_id = %d, want 2", result.TaskID)
	}
	if len(result.Upstream) != 1 || result.Upstream[0].ID != 1 {
		t.Errorf("upstream = %v, want [#1]", result.Upstream)
	}
	if len(result.Downstream) != 1 || result.Downstream[0].ID != 3 {
		t.Errorf("downstream = %v, want [#3]", result.Downstream)
	}
}

func TestDepsUpstreamOnly(t *testing.T) {
	dir := setupDepsBoard(t)

	var result struct {
		Upstream   []struct{ ID int } `json:"upstream"`
		Downstream []struct{ ID int } `json:"downstream"`
	}
	runKanbanJSON(t, dir, &result, "deps", "3", "--upstream")
	if len(result.Upstream) != 1 || result.Upstream[0].ID != 2 {
		t.Errorf("upstream = %v, want [#2]", result.Upstream)
	}
	if result.Downstream != nil {
		t.Errorf("downstream should be omitted, got %v", result.Downstream)
	}
}

func TestDepsDownstreamOnly(t *testing.T) {
	dir := setupDepsBoard(t)

	var result struct {
		Upstream   []struct{ ID int } `json:"upstream"`
		Downstream []struct{ ID int } `json:"downstream"`
	}
	runKanbanJSON(t, dir, &result, "deps", "1", "--downstream")
	if len(result.Downstream) != 1 || result.Downstream[0].ID != 2 {
		t.Errorf("downstream = %v, want [#2]", result.Downstream)
	}
	if result.Upstream != nil {
		t.Errorf("upstream should be omitted, got %v", result.Upstream)
	}
}

func TestDepsTransitiveUpstream(t *testing.T) {
	dir := setupDepsBoard(t)

	var result struct {
		Upstream []struct{ ID int } `json:"upstream"`
	}
	runKanbanJSON(t, dir, &result, "deps", "3", "--upstream", "--transitive")
	if len(result.Upstream) != 2 {
		t.Fatalf("transitive upstream count = %d, want 2", len(result.Upstream))
	}
	ids := map[int]bool{}
	for _, u := range result.Upstream {
		ids[u.ID] = true
	}
	if !ids[1] || !ids[2] {
		t.Errorf("transitive upstream IDs = %v, want {1, 2}", ids)
	}
}

func TestDepsTransitiveDownstream(t *testing.T) {
	dir := setupDepsBoard(t)

	var result struct {
		Downstream []struct{ ID int } `json:"downstream"`
	}
	runKanbanJSON(t, dir, &result, "deps", "1", "--downstream", "--transitive")
	if len(result.Downstream) != 2 {
		t.Fatalf("transitive downstream count = %d, want 2", len(result.Downstream))
	}
	ids := map[int]bool{}
	for _, d := range result.Downstream {
		ids[d.ID] = true
	}
	if !ids[2] || !ids[3] {
		t.Errorf("transitive downstream IDs = %v, want {2, 3}", ids)
	}
}

func TestDepsIsolatedTask(t *testing.T) {
	dir := setupDepsBoard(t)

	var result struct {
		Upstream   []struct{ ID int } `json:"upstream"`
		Downstream []struct{ ID int } `json:"downstream"`
	}
	runKanbanJSON(t, dir, &result, "deps", "4")
	if len(result.Upstream) != 0 {
		t.Errorf("upstream for isolated task = %v, want empty", result.Upstream)
	}
	if len(result.Downstream) != 0 {
		t.Errorf("downstream for isolated task = %v, want empty", result.Downstream)
	}
}

func TestDepsNonexistentTask(t *testing.T) {
	dir := setupDepsBoard(t)

	errResp := runKanbanJSONError(t, dir, "deps", "99")
	if errResp.Code != "TASK_NOT_FOUND" {
		t.Errorf("code = %q, want TASK_NOT_FOUND", errResp.Code)
	}
}

func TestDepsJSONContainsStatus(t *testing.T) {
	dir := setupDepsBoard(t)

	var raw json.RawMessage
	runKanbanJSON(t, dir, &raw, "deps", "2")

	var result struct {
		Upstream []struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
			Title  string `json:"title"`
		} `json:"upstream"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if len(result.Upstream) != 1 {
		t.Fatalf("upstream count = %d, want 1", len(result.Upstream))
	}
	if result.Upstream[0].Status != "done" {
		t.Errorf("upstream status = %q, want done", result.Upstream[0].Status)
	}
	if result.Upstream[0].Title != "Base task" {
		t.Errorf("upstream title = %q, want 'Base task'", result.Upstream[0].Title)
	}
}

func TestDepsCompactOutput(t *testing.T) {
	dir := setupDepsBoard(t)

	r := runKanban(t, dir, "--compact", "deps", "2")
	if r.exitCode != 0 {
		t.Fatalf("deps --compact failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "#2") {
		t.Errorf("compact output missing task ID, got: %s", r.stdout)
	}
	if !strings.Contains(r.stdout, "upstream:") {
		t.Errorf("compact output missing upstream section, got: %s", r.stdout)
	}
	if !strings.Contains(r.stdout, "downstream:") {
		t.Errorf("compact output missing downstream section, got: %s", r.stdout)
	}
}

func TestDepsTableOutput(t *testing.T) {
	dir := setupDepsBoard(t)

	r := runKanban(t, dir, "deps", "2")
	if r.exitCode != 0 {
		t.Fatalf("deps table failed: %s", r.stderr)
	}
	if !strings.Contains(r.stdout, "Dependencies for task #2") {
		t.Errorf("table output missing header, got: %s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Upstream") {
		t.Errorf("table output missing Upstream section, got: %s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Downstream") {
		t.Errorf("table output missing Downstream section, got: %s", r.stdout)
	}
}

// Test that --unblocked --status backlog works correctly with deps.
func TestListUnblockedStatusBacklog(t *testing.T) {
	dir := setupDepsBoard(t)

	// Task 4 (Isolated) is in backlog with no deps → should be unblocked.
	// Task 3 (Leaf) is in backlog but depends on task 2 which is in-progress → should be blocked.
	var tasks []struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		Status    string `json:"status"`
		DependsOn []int  `json:"depends_on"`
	}
	runKanbanJSON(t, dir, &tasks, "list", "--unblocked", "--status", "backlog")

	// Task 3 depends on task 2 (in-progress, not terminal) → should be blocked.
	// Task 4 has no deps → should be unblocked.
	// Only task 4 should appear.
	if len(tasks) != 1 {
		for _, tk := range tasks {
			t.Logf("  got: #%d %q status=%s deps=%v", tk.ID, tk.Title, tk.Status, tk.DependsOn)
		}
		t.Fatalf("unblocked backlog count = %d, want 1", len(tasks))
	}
	if tasks[0].ID != 4 {
		t.Errorf("unblocked backlog task = #%d, want #4", tasks[0].ID)
	}
}
