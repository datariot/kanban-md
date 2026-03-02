package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/filelock"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// importSpec is the input format for bulk task import.
type importSpec struct {
	Parent *importParent `json:"parent" yaml:"parent"`
	Tasks  []importTask  `json:"tasks" yaml:"tasks"`
}

type importParent struct {
	Title    string   `json:"title" yaml:"title"`
	Priority string   `json:"priority,omitempty" yaml:"priority,omitempty"`
	Tags     []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Status   string   `json:"status,omitempty" yaml:"status,omitempty"`
	Body     string   `json:"body,omitempty" yaml:"body,omitempty"`
}

type importTask struct {
	Ref       string   `json:"ref" yaml:"ref"`
	Title     string   `json:"title" yaml:"title"`
	Priority  string   `json:"priority,omitempty" yaml:"priority,omitempty"`
	Tags      []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Body      string   `json:"body,omitempty" yaml:"body,omitempty"`
	DependsOn []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Status    string   `json:"status,omitempty" yaml:"status,omitempty"`
	Assignee  string   `json:"assignee,omitempty" yaml:"assignee,omitempty"`
}

// importResult is a single entry in the ref-to-ID mapping output.
type importResult struct {
	Ref string `json:"ref"`
	ID  int    `json:"id"`
}

// importOutput is the JSON output for the import command.
type importOutput struct {
	Parent  *int           `json:"parent,omitempty"`
	Created int            `json:"created"`
	Mapping []importResult `json:"mapping"`
}

var importCmd = &cobra.Command{
	Use:   "import [FILE]",
	Short: "Bulk-create tasks from a JSON or YAML spec",
	Long: `Creates multiple tasks at once from a structured input file.

Tasks can reference each other via local "ref" IDs, and dependencies
are wired automatically by mapping refs to created kanban IDs.

Accepts JSON or YAML input from a file argument or stdin (use "-" for stdin).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func runImport(_ *cobra.Command, args []string) error {
	data, err := readImportInput(args)
	if err != nil {
		return err
	}

	spec, err := parseImportSpec(data)
	if err != nil {
		return err
	}

	if err = validateImportSpec(spec); err != nil {
		return err
	}

	dir, err := resolveDir()
	if err != nil {
		return err
	}
	unlock, err := filelock.Lock(filepath.Join(dir, ".lock"))
	if err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer unlock() //nolint:errcheck // best-effort unlock on exit

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	now := time.Now()
	var parentID *int

	if spec.Parent != nil {
		var pid int
		pid, err = createImportParent(cfg, spec.Parent, now)
		if err != nil {
			return err
		}
		parentID = &pid
	}

	mapping, err := createImportTasks(cfg, spec.Tasks, parentID, now)
	if err != nil {
		return err
	}

	if err = cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	return outputImportResult(parentID, mapping)
}

// createImportParent creates the parent task and returns its ID. It increments cfg.NextID.
func createImportParent(cfg *config.Config, p *importParent, now time.Time) (int, error) {
	pt := &task.Task{
		ID:       cfg.NextID,
		Title:    p.Title,
		Status:   cfg.Defaults.Status,
		Priority: cfg.Defaults.Priority,
		Class:    cfg.Defaults.Class,
		Created:  now,
		Updated:  now,
	}
	if p.Priority != "" {
		if err := task.ValidatePriority(p.Priority, cfg.Priorities); err != nil {
			return 0, fmt.Errorf("parent: %w", err)
		}
		pt.Priority = p.Priority
	}
	if p.Status != "" {
		if err := task.ValidateStatus(p.Status, cfg.StatusNames()); err != nil {
			return 0, fmt.Errorf("parent: %w", err)
		}
		pt.Status = p.Status
	}
	if len(p.Tags) > 0 {
		pt.Tags = p.Tags
	}
	if p.Body != "" {
		pt.Body = p.Body
	}

	slug := task.GenerateSlug(pt.Title)
	filename := task.GenerateFilename(pt.ID, slug)
	path := filepath.Join(cfg.TasksPath(), filename)
	pt.File = path

	if err := task.Write(path, pt); err != nil {
		return 0, fmt.Errorf("writing parent task: %w", err)
	}

	id := cfg.NextID
	logActivity(cfg, "create", pt.ID, pt.Title)
	cfg.NextID++
	return id, nil
}

// createImportTasks creates child tasks, wires dependencies, and returns the ref-to-ID mapping.
// It increments cfg.NextID for each task created.
func createImportTasks(cfg *config.Config, tasks []importTask, parentID *int, now time.Time) ([]importResult, error) {
	refMap := make(map[string]int, len(tasks))
	mapping := make([]importResult, 0, len(tasks))

	for i := range tasks {
		st := &tasks[i]

		t := &task.Task{
			ID:       cfg.NextID,
			Title:    st.Title,
			Status:   cfg.Defaults.Status,
			Priority: cfg.Defaults.Priority,
			Class:    cfg.Defaults.Class,
			Created:  now,
			Updated:  now,
			Parent:   parentID,
		}

		if err := applyImportTaskFields(t, st, cfg); err != nil {
			return nil, err
		}

		// Resolve depends_on refs to kanban IDs.
		for _, depRef := range st.DependsOn {
			depID, ok := refMap[depRef]
			if !ok {
				return nil, clierr.Newf(clierr.InvalidInput,
					"task %q depends on unknown ref %q (refs must be declared before use)", st.Ref, depRef)
			}
			t.DependsOn = append(t.DependsOn, depID)
		}

		slug := task.GenerateSlug(t.Title)
		filename := task.GenerateFilename(t.ID, slug)
		path := filepath.Join(cfg.TasksPath(), filename)
		t.File = path

		if err := task.Write(path, t); err != nil {
			return nil, fmt.Errorf("writing task %q: %w", st.Ref, err)
		}

		refMap[st.Ref] = cfg.NextID
		mapping = append(mapping, importResult{Ref: st.Ref, ID: cfg.NextID})
		logActivity(cfg, "create", t.ID, t.Title)
		cfg.NextID++
	}

	return mapping, nil
}

// applyImportTaskFields sets optional fields on a task from the import spec.
func applyImportTaskFields(t *task.Task, st *importTask, cfg *config.Config) error {
	if st.Priority != "" {
		if err := task.ValidatePriority(st.Priority, cfg.Priorities); err != nil {
			return fmt.Errorf("task %q: %w", st.Ref, err)
		}
		t.Priority = st.Priority
	}
	if st.Status != "" {
		if err := task.ValidateStatus(st.Status, cfg.StatusNames()); err != nil {
			return fmt.Errorf("task %q: %w", st.Ref, err)
		}
		t.Status = st.Status
	}
	if len(st.Tags) > 0 {
		t.Tags = st.Tags
	}
	if st.Body != "" {
		t.Body = st.Body
	}
	if st.Assignee != "" {
		t.Assignee = st.Assignee
	}
	return nil
}

// outputImportResult prints the import results in the appropriate format.
func outputImportResult(parentID *int, mapping []importResult) error {
	out := importOutput{
		Parent:  parentID,
		Created: len(mapping),
		Mapping: mapping,
	}

	if outputFormat() == output.FormatJSON {
		return output.JSON(os.Stdout, out)
	}

	if parentID != nil {
		output.Messagef(os.Stdout, "Created %d tasks under parent #%d", len(mapping), *parentID)
	} else {
		output.Messagef(os.Stdout, "Created %d tasks", len(mapping))
	}
	for _, m := range mapping {
		output.Messagef(os.Stdout, "  %s -> #%d", m.Ref, m.ID)
	}
	return nil
}

// readImportInput reads from the given file argument or stdin.
func readImportInput(args []string) ([]byte, error) {
	if len(args) == 0 || args[0] == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		if len(data) == 0 {
			return nil, clierr.New(clierr.InvalidInput, "no input provided on stdin")
		}
		return data, nil
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", args[0], err)
	}
	return data, nil
}

// parseImportSpec tries JSON first, then YAML.
func parseImportSpec(data []byte) (*importSpec, error) {
	var spec importSpec
	if json.Valid(data) {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, clierr.Newf(clierr.InvalidInput, "invalid JSON: %v", err)
		}
		return &spec, nil
	}

	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, clierr.Newf(clierr.InvalidInput, "invalid YAML: %v", err)
	}
	return &spec, nil
}

// validateImportSpec checks that the spec is well-formed before creating tasks.
func validateImportSpec(spec *importSpec) error {
	if len(spec.Tasks) == 0 {
		return clierr.New(clierr.InvalidInput, "import spec must contain at least one task")
	}

	if spec.Parent != nil && strings.TrimSpace(spec.Parent.Title) == "" {
		return clierr.New(clierr.InvalidInput, "parent task title is required")
	}

	refs := make(map[string]bool, len(spec.Tasks))
	for i, t := range spec.Tasks {
		if strings.TrimSpace(t.Ref) == "" {
			return clierr.Newf(clierr.InvalidInput, "task[%d]: ref is required", i)
		}
		if refs[t.Ref] {
			return clierr.Newf(clierr.InvalidInput, "duplicate ref %q", t.Ref)
		}
		refs[t.Ref] = true

		if strings.TrimSpace(t.Title) == "" {
			return clierr.Newf(clierr.InvalidInput, "task %q: title is required", t.Ref)
		}

		for _, dep := range t.DependsOn {
			if dep == t.Ref {
				return clierr.Newf(clierr.InvalidInput, "task %q: self-referencing dependency", t.Ref)
			}
		}
	}

	for _, t := range spec.Tasks {
		for _, dep := range t.DependsOn {
			if !refs[dep] {
				return clierr.Newf(clierr.InvalidInput,
					"task %q: depends_on ref %q does not match any task in the spec", t.Ref, dep)
			}
		}
	}

	return nil
}
