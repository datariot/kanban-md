package board

import (
	"github.com/antopolskiy/kanban-md/internal/task"
)

// DepDirection indicates which direction to traverse the dependency graph.
type DepDirection int

const (
	// DepBoth traverses both upstream and downstream.
	DepBoth DepDirection = iota
	// DepUpstream traverses only upstream (what this task depends on).
	DepUpstream
	// DepDownstream traverses only downstream (what depends on this task).
	DepDownstream
)

// DepResult holds the result of a dependency query for a single task.
type DepResult struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// DepsOutput is the structured output of the deps command.
type DepsOutput struct {
	TaskID     int          `json:"task_id"`
	TaskTitle  string       `json:"task_title"`
	Upstream   []*DepResult `json:"upstream,omitempty"`
	Downstream []*DepResult `json:"downstream,omitempty"`
}

// Deps computes upstream and/or downstream dependencies for a given task.
// If transitive is true, follows the full ancestor/descendant chain.
func Deps(allTasks []*task.Task, targetID int, direction DepDirection, transitive bool) *DepsOutput {
	byID := make(map[int]*task.Task, len(allTasks))
	for _, t := range allTasks {
		byID[t.ID] = t
	}

	target := byID[targetID]
	out := &DepsOutput{
		TaskID:    targetID,
		TaskTitle: target.Title,
	}

	if direction == DepBoth || direction == DepUpstream {
		if transitive {
			out.Upstream = transitiveUpstream(targetID, byID)
		} else {
			out.Upstream = directUpstream(target, byID)
		}
	}

	if direction == DepBoth || direction == DepDownstream {
		// Build reverse index: task ID -> list of tasks that depend on it.
		downstream := buildDownstreamIndex(allTasks)
		if transitive {
			out.Downstream = transitiveDownstream(targetID, downstream, byID)
		} else {
			out.Downstream = directDownstream(targetID, downstream, byID)
		}
	}

	return out
}

// directUpstream returns the immediate dependencies of the target task.
func directUpstream(target *task.Task, byID map[int]*task.Task) []*DepResult {
	var results []*DepResult
	for _, depID := range target.DependsOn {
		if t, ok := byID[depID]; ok {
			results = append(results, &DepResult{ID: t.ID, Title: t.Title, Status: t.Status})
		}
	}
	return results
}

// transitiveUpstream follows the full ancestor chain via BFS.
func transitiveUpstream(startID int, byID map[int]*task.Task) []*DepResult {
	visited := map[int]bool{startID: true}
	queue := []int{startID}
	var results []*DepResult

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		t, ok := byID[current]
		if !ok {
			continue
		}
		for _, depID := range t.DependsOn {
			if visited[depID] {
				continue
			}
			visited[depID] = true
			if dep, ok := byID[depID]; ok {
				results = append(results, &DepResult{ID: dep.ID, Title: dep.Title, Status: dep.Status})
				queue = append(queue, depID)
			}
		}
	}
	return results
}

// buildDownstreamIndex builds a map from task ID to list of task IDs that depend on it.
func buildDownstreamIndex(allTasks []*task.Task) map[int][]int {
	downstream := make(map[int][]int)
	for _, t := range allTasks {
		for _, depID := range t.DependsOn {
			downstream[depID] = append(downstream[depID], t.ID)
		}
	}
	return downstream
}

// directDownstream returns tasks that directly depend on the target.
func directDownstream(targetID int, downstream map[int][]int, byID map[int]*task.Task) []*DepResult {
	var results []*DepResult
	for _, id := range downstream[targetID] {
		if t, ok := byID[id]; ok {
			results = append(results, &DepResult{ID: t.ID, Title: t.Title, Status: t.Status})
		}
	}
	return results
}

// transitiveDownstream follows the full descendant chain via BFS.
func transitiveDownstream(startID int, downstream map[int][]int, byID map[int]*task.Task) []*DepResult {
	visited := map[int]bool{startID: true}
	queue := []int{startID}
	var results []*DepResult

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, id := range downstream[current] {
			if visited[id] {
				continue
			}
			visited[id] = true
			if t, ok := byID[id]; ok {
				results = append(results, &DepResult{ID: t.ID, Title: t.Title, Status: t.Status})
				queue = append(queue, id)
			}
		}
	}
	return results
}
