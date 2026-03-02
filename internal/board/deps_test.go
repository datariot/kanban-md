package board

import (
	"testing"

	"github.com/antopolskiy/kanban-md/internal/task"
)

func makeDepsTasks() []*task.Task {
	// Task graph:
	//   1 -> 2 -> 4
	//   1 -> 3 -> 4
	//   5 (isolated)
	return []*task.Task{
		{ID: 1, Title: "Task 1", Status: "done"},
		{ID: 2, Title: "Task 2", Status: "in-progress", DependsOn: []int{1}},
		{ID: 3, Title: "Task 3", Status: "todo", DependsOn: []int{1}},
		{ID: 4, Title: "Task 4", Status: "backlog", DependsOn: []int{2, 3}},
		{ID: 5, Title: "Task 5", Status: "backlog"},
	}
}

func TestDepsDirectUpstream(t *testing.T) {
	tasks := makeDepsTasks()
	result := Deps(tasks, 4, DepUpstream, false)

	if result.TaskID != 4 {
		t.Errorf("TaskID = %d, want 4", result.TaskID)
	}
	if len(result.Upstream) != 2 {
		t.Fatalf("upstream count = %d, want 2", len(result.Upstream))
	}
	ids := map[int]bool{}
	for _, d := range result.Upstream {
		ids[d.ID] = true
	}
	if !ids[2] || !ids[3] {
		t.Errorf("upstream IDs = %v, want {2, 3}", ids)
	}
	if result.Downstream != nil {
		t.Errorf("downstream should be nil for upstream-only query, got %v", result.Downstream)
	}
}

func TestDepsDirectDownstream(t *testing.T) {
	tasks := makeDepsTasks()
	result := Deps(tasks, 1, DepDownstream, false)

	if len(result.Downstream) != 2 {
		t.Fatalf("downstream count = %d, want 2", len(result.Downstream))
	}
	ids := map[int]bool{}
	for _, d := range result.Downstream {
		ids[d.ID] = true
	}
	if !ids[2] || !ids[3] {
		t.Errorf("downstream IDs = %v, want {2, 3}", ids)
	}
	if result.Upstream != nil {
		t.Errorf("upstream should be nil for downstream-only query, got %v", result.Upstream)
	}
}

func TestDepsTransitiveUpstream(t *testing.T) {
	tasks := makeDepsTasks()
	result := Deps(tasks, 4, DepUpstream, true)

	if len(result.Upstream) != 3 {
		t.Fatalf("transitive upstream count = %d, want 3", len(result.Upstream))
	}
	ids := map[int]bool{}
	for _, d := range result.Upstream {
		ids[d.ID] = true
	}
	if !ids[1] || !ids[2] || !ids[3] {
		t.Errorf("transitive upstream IDs = %v, want {1, 2, 3}", ids)
	}
}

func TestDepsTransitiveDownstream(t *testing.T) {
	tasks := makeDepsTasks()
	result := Deps(tasks, 1, DepDownstream, true)

	if len(result.Downstream) != 3 {
		t.Fatalf("transitive downstream count = %d, want 3", len(result.Downstream))
	}
	ids := map[int]bool{}
	for _, d := range result.Downstream {
		ids[d.ID] = true
	}
	if !ids[2] || !ids[3] || !ids[4] {
		t.Errorf("transitive downstream IDs = %v, want {2, 3, 4}", ids)
	}
}

func TestDepsBothDirections(t *testing.T) {
	tasks := makeDepsTasks()
	result := Deps(tasks, 2, DepBoth, false)

	if len(result.Upstream) != 1 || result.Upstream[0].ID != 1 {
		t.Errorf("upstream = %v, want [#1]", result.Upstream)
	}
	if len(result.Downstream) != 1 || result.Downstream[0].ID != 4 {
		t.Errorf("downstream = %v, want [#4]", result.Downstream)
	}
}

func TestDepsIsolatedTask(t *testing.T) {
	tasks := makeDepsTasks()
	result := Deps(tasks, 5, DepBoth, false)

	if len(result.Upstream) != 0 {
		t.Errorf("upstream = %v, want empty", result.Upstream)
	}
	if len(result.Downstream) != 0 {
		t.Errorf("downstream = %v, want empty", result.Downstream)
	}
}

func TestDepsNoCycleInfiniteLoop(t *testing.T) {
	// Create a cycle: 1 -> 2 -> 3 -> 1
	tasks := []*task.Task{
		{ID: 1, Title: "A", Status: "backlog", DependsOn: []int{3}},
		{ID: 2, Title: "B", Status: "backlog", DependsOn: []int{1}},
		{ID: 3, Title: "C", Status: "backlog", DependsOn: []int{2}},
	}
	// Should not hang.
	result := Deps(tasks, 1, DepUpstream, true)
	if len(result.Upstream) != 2 {
		t.Errorf("transitive upstream in cycle = %d, want 2", len(result.Upstream))
	}
}
