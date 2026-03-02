package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show flow metrics",
	Long:  `Displays flow metrics: throughput, average lead/cycle time, flow efficiency, and aging work items.`,
	RunE:  runMetrics,
}

func init() {
	metricsCmd.Flags().String("since", "", "only include tasks completed after this date (YYYY-MM-DD)")
	metricsCmd.Flags().Int("parent", 0, "scope metrics to children of a parent task ID")
	rootCmd.AddCommand(metricsCmd)
}

func runMetrics(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	tasks, err := loadMetricsTasks(cmd, cfg)
	if err != nil {
		return err
	}

	now := time.Now()
	m := board.ComputeMetrics(cfg, tasks, now)

	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(os.Stdout, m)
	}
	if format == output.FormatCompact {
		output.MetricsCompact(os.Stdout, m)
		return nil
	}

	output.MetricsTable(os.Stdout, m)
	return nil
}

// loadMetricsTasks loads and filters tasks for metrics computation.
func loadMetricsTasks(cmd *cobra.Command, cfg *config.Config) ([]*task.Task, error) {
	allTasks, warnings, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return nil, err
	}
	printWarnings(warnings)
	if allTasks == nil {
		allTasks = []*task.Task{}
	}

	// Exclude archived tasks from metrics.
	tasks := make([]*task.Task, 0, len(allTasks))
	for _, t := range allTasks {
		if !cfg.IsArchivedStatus(t.Status) {
			tasks = append(tasks, t)
		}
	}

	// Filter to children of a parent task.
	if cmd.Flags().Changed("parent") {
		parentID, _ := cmd.Flags().GetInt("parent")
		filtered := make([]*task.Task, 0, len(tasks))
		for _, t := range tasks {
			if t.Parent != nil && *t.Parent == parentID {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	sinceStr, _ := cmd.Flags().GetString("since")
	if sinceStr != "" {
		d, parseErr := date.Parse(sinceStr)
		if parseErr != nil {
			return nil, task.ValidateDate("since", sinceStr, parseErr)
		}
		sinceTime := d.Time
		filtered := make([]*task.Task, 0, len(tasks))
		for _, t := range tasks {
			if t.Completed == nil || t.Completed.After(sinceTime) {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	return tasks, nil
}
