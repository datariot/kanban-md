package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var showCmd = &cobra.Command{
	Use:   "show ID[,ID,...]",
	Short: "Show task details",
	Long:  `Displays full details of one or more tasks including their markdown body.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	showCmd.Flags().String("section", "", "extract a specific named section from the body")
	showCmd.Flags().Bool("prompt", false, "token-efficient output for LLM prompts")
	showCmd.Flags().String("fields", "", "fields to include in --prompt output (comma-separated: title,status,tags,body)")
	showCmd.Flags().Bool("children", false, "show parent task with all children's status summary")
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	promptMode, _ := cmd.Flags().GetBool("prompt")
	fieldsSpec, _ := cmd.Flags().GetString("fields")
	childrenMode, _ := cmd.Flags().GetBool("children")

	// Parse IDs (supports comma-separated).
	ids, err := parseIDs(args[0])
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// --children mode: show parent + children summary.
	if childrenMode {
		if len(ids) > 1 {
			return clierr.New(clierr.InvalidInput, "--children only supports a single task ID")
		}
		return runShowChildren(cfg, ids[0])
	}

	// Load all requested tasks.
	tasks := make([]*task.Task, 0, len(ids))
	for _, id := range ids {
		path, pathErr := task.FindByID(cfg.TasksPath(), id)
		if pathErr != nil {
			return pathErr
		}
		t, readErr := task.Read(path)
		if readErr != nil {
			return readErr
		}
		tasks = append(tasks, t)
	}

	// --prompt mode.
	if promptMode {
		fields, fieldErr := output.ParsePromptFields(fieldsSpec)
		if fieldErr != nil {
			return clierr.New(clierr.InvalidInput, fieldErr.Error())
		}
		output.TasksPrompt(os.Stdout, tasks, fields)
		return nil
	}

	// For non-prompt modes, only single task is supported.
	if len(tasks) > 1 {
		return clierr.New(clierr.InvalidInput, "multiple IDs only supported with --prompt")
	}

	t := tasks[0]

	sectionName, _ := cmd.Flags().GetString("section")
	if sectionName != "" {
		return outputSection(t, sectionName)
	}

	return outputTaskDetail(t)
}

// ChildrenSummary holds a parent task and its children for JSON output.
type ChildrenSummary struct {
	Parent   *task.Task     `json:"parent"`
	Children []*task.Task   `json:"children"`
	Counts   map[string]int `json:"status_counts"`
}

func runShowChildren(cfg *config.Config, parentID int) error {
	// Load parent task.
	parentPath, err := task.FindByID(cfg.TasksPath(), parentID)
	if err != nil {
		return err
	}
	parent, err := task.Read(parentPath)
	if err != nil {
		return err
	}

	// Load all tasks and filter children.
	allTasks, warnings, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return err
	}
	printWarnings(warnings)

	var children []*task.Task
	for _, t := range allTasks {
		if t.Parent != nil && *t.Parent == parentID {
			children = append(children, t)
		}
	}

	// Count statuses.
	counts := make(map[string]int)
	for _, c := range children {
		counts[c.Status]++
	}

	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(os.Stdout, ChildrenSummary{
			Parent:   parent,
			Children: children,
			Counts:   counts,
		})
	}
	if format == output.FormatCompact {
		outputChildrenCompact(parent, children, counts)
		return nil
	}

	outputChildrenTable(parent, children, counts)
	return nil
}

func outputChildrenTable(parent *task.Task, children []*task.Task, counts map[string]int) {
	output.TaskDetail(os.Stdout, parent)

	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Children: %d tasks", len(children))
	if len(counts) > 0 {
		fmt.Fprint(os.Stdout, " (")
		first := true
		for status, count := range counts {
			if !first {
				fmt.Fprint(os.Stdout, ", ")
			}
			fmt.Fprintf(os.Stdout, "%s: %d", status, count)
			first = false
		}
		fmt.Fprint(os.Stdout, ")")
	}
	fmt.Fprintln(os.Stdout)

	if len(children) > 0 {
		fmt.Fprintln(os.Stdout)
		output.TaskTable(os.Stdout, children)
	}
}

func outputChildrenCompact(parent *task.Task, children []*task.Task, counts map[string]int) {
	// Parent on first line.
	fmt.Fprintf(os.Stdout, "#%d [%s/%s] %s (%d children)\n",
		parent.ID, parent.Status, parent.Priority, parent.Title, len(children))

	// Status summary.
	if len(counts) > 0 {
		fmt.Fprint(os.Stdout, "  Status:")
		for status, count := range counts {
			fmt.Fprintf(os.Stdout, " %s=%d", status, count)
		}
		fmt.Fprintln(os.Stdout)
	}

	// One line per child.
	for _, c := range children {
		fmt.Fprintf(os.Stdout, "  #%d [%s] %s\n",
			c.ID, c.Status, c.Title)
	}
}

func outputSection(t *task.Task, sectionName string) error {
	content, ok := task.GetSection(t.Body, sectionName)
	if !ok {
		return clierr.New(clierr.InvalidInput, fmt.Sprintf("section %q not found", sectionName))
	}

	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(os.Stdout, map[string]string{
			"section": sectionName,
			"content": content,
		})
	}

	fmt.Fprintln(os.Stdout, content)
	return nil
}

func outputTaskDetail(t *task.Task) error {
	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(os.Stdout, t)
	}
	if format == output.FormatCompact {
		output.TaskDetailCompact(os.Stdout, t)
		return nil
	}

	output.TaskDetail(os.Stdout, t)
	return nil
}
