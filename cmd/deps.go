package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/output"
	"github.com/antopolskiy/kanban-md/internal/task"
)

var depsCmd = &cobra.Command{
	Use:   "deps ID",
	Short: "Show task dependencies",
	Long:  `Displays upstream (what this task depends on) and downstream (what depends on this task) dependencies.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDeps,
}

func init() {
	depsCmd.Flags().Bool("upstream", false, "show only upstream dependencies (what this task needs)")
	depsCmd.Flags().Bool("downstream", false, "show only downstream dependencies (what this task unblocks)")
	depsCmd.Flags().Bool("transitive", false, "follow the full dependency chain, not just direct deps")
	rootCmd.AddCommand(depsCmd)
}

func runDeps(cmd *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return task.ValidateTaskID(args[0])
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Verify the task exists.
	if _, findErr := task.FindByID(cfg.TasksPath(), id); findErr != nil {
		return findErr
	}

	allTasks, warnings, err := task.ReadAllLenient(cfg.TasksPath())
	if err != nil {
		return err
	}
	printWarnings(warnings)

	upstream, _ := cmd.Flags().GetBool("upstream")
	downstream, _ := cmd.Flags().GetBool("downstream")
	transitive, _ := cmd.Flags().GetBool("transitive")

	direction := board.DepBoth
	if upstream && !downstream {
		direction = board.DepUpstream
	} else if downstream && !upstream {
		direction = board.DepDownstream
	}

	result := board.Deps(allTasks, id, direction, transitive)
	return outputDeps(result, direction)
}

func outputDeps(result *board.DepsOutput, direction board.DepDirection) error {
	format := outputFormat()
	if format == output.FormatJSON {
		return output.JSON(os.Stdout, result)
	}

	if format == output.FormatCompact {
		outputDepsCompact(result, direction)
		return nil
	}

	outputDepsTable(result, direction)
	return nil
}

func outputDepsTable(result *board.DepsOutput, direction board.DepDirection) {
	fmt.Fprintf(os.Stdout, "Dependencies for task #%d: %s\n", result.TaskID, result.TaskTitle)
	fmt.Fprintln(os.Stdout, strings.Repeat("─", 50)) //nolint:mnd // separator width

	if direction == board.DepBoth || direction == board.DepUpstream {
		fmt.Fprintln(os.Stdout, "\nUpstream (this task depends on):")
		if len(result.Upstream) == 0 {
			fmt.Fprintln(os.Stdout, "  (none)")
		} else {
			for _, d := range result.Upstream {
				fmt.Fprintf(os.Stdout, "  #%-4d [%-12s] %s\n", d.ID, d.Status, d.Title)
			}
		}
	}

	if direction == board.DepBoth || direction == board.DepDownstream {
		fmt.Fprintln(os.Stdout, "\nDownstream (depends on this task):")
		if len(result.Downstream) == 0 {
			fmt.Fprintln(os.Stdout, "  (none)")
		} else {
			for _, d := range result.Downstream {
				fmt.Fprintf(os.Stdout, "  #%-4d [%-12s] %s\n", d.ID, d.Status, d.Title)
			}
		}
	}
}

func outputDepsCompact(result *board.DepsOutput, direction board.DepDirection) {
	fmt.Fprintf(os.Stdout, "#%d %s\n", result.TaskID, result.TaskTitle)

	if direction == board.DepBoth || direction == board.DepUpstream {
		if len(result.Upstream) == 0 {
			fmt.Fprintln(os.Stdout, "  upstream: (none)")
		} else {
			ids := make([]string, len(result.Upstream))
			for i, d := range result.Upstream {
				ids[i] = fmt.Sprintf("#%d[%s]", d.ID, d.Status)
			}
			fmt.Fprintf(os.Stdout, "  upstream: %s\n", strings.Join(ids, " "))
		}
	}

	if direction == board.DepBoth || direction == board.DepDownstream {
		if len(result.Downstream) == 0 {
			fmt.Fprintln(os.Stdout, "  downstream: (none)")
		} else {
			ids := make([]string, len(result.Downstream))
			for i, d := range result.Downstream {
				ids[i] = fmt.Sprintf("#%d[%s]", d.ID, d.Status)
			}
			fmt.Fprintf(os.Stdout, "  downstream: %s\n", strings.Join(ids, " "))
		}
	}
}
