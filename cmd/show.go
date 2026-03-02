package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var showCmd = &cobra.Command{
	Use:   "show ID",
	Short: "Show task details",
	Long:  `Displays full details of a single task including its markdown body.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	showCmd.Flags().String("section", "", "extract a specific named section from the body")
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return task.ValidateTaskID(args[0])
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	path, err := task.FindByID(cfg.TasksPath(), id)
	if err != nil {
		return err
	}

	t, err := task.Read(path)
	if err != nil {
		return err
	}

	sectionName, _ := cmd.Flags().GetString("section")
	if sectionName != "" {
		return outputSection(t, sectionName)
	}

	return outputTaskDetail(t)
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
