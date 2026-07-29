package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/carlosprados/og-cli/internal/output"
	"github.com/carlosprados/og-cli/pkg/opengate"
	"github.com/spf13/cobra"
)

// --- jobs ---

var jobsCmd = &cobra.Command{
	Use:     "jobs",
	Aliases: []string{"job"},
	Short:   "Manage OpenGate operation jobs",
}

var jobsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search jobs",
	RunE:  runJobsSearch,
}

var (
	jobSearchFilter string
	jobSearchWhere  []string
	jobSearchLimit  int
)

func runJobsSearch(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := opengate.New(p.Host, p.Token)

	filter, err := buildSearchFilter(jobSearchWhere, jobSearchLimit, nil, jobSearchFilter)
	if err != nil {
		return err
	}

	resp, err := c.SearchJobs(cmd.Context(), filter)
	if err != nil {
		return err
	}

	if outFmt == output.FormatJSON {
		return output.PrintJSON(os.Stdout, resp.Jobs)
	}

	rows := make([][]string, len(resp.Jobs))
	for i, raw := range resp.Jobs {
		var m map[string]any
		json.Unmarshal(raw, &m)
		id, _ := m["id"].(string)
		name := ""
		status := ""
		if req, ok := m["request"].(map[string]any); ok {
			name, _ = req["name"].(string)
		}
		if rep, ok := m["report"].(map[string]any); ok {
			if sum, ok := rep["summary"].(map[string]any); ok {
				status, _ = sum["status"].(string)
			}
		}
		rows[i] = []string{id, name, status}
	}

	output.PrintTable(os.Stdout, []string{"ID", "Operation", "Status"}, rows)
	return nil
}

var jobsGetCmd = &cobra.Command{
	Use:   "get <job-id>",
	Short: "Get a job report",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)

		data, err := c.GetJob(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		return output.PrintJSON(os.Stdout, json.RawMessage(data))
	},
}

var jobsCreateCmd = &cobra.Command{
	Use:   "create -f <file.json>",
	Short: "Create a new operation job",
	RunE:  runJobsCreate,
}

var jobCreateFile string

func runJobsCreate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	body, err := os.ReadFile(jobCreateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := opengate.New(p.Host, p.Token)
	resp, err := c.CreateJob(cmd.Context(), body)
	if err != nil {
		return err
	}

	return output.PrintJSON(os.Stdout, json.RawMessage(resp))
}

var jobsCancelCmd = &cobra.Command{
	Use:   "cancel <job-id>",
	Short: "Cancel a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := confirmDestructive(fmt.Sprintf("cancel job %q", args[0])); err != nil {
			return err
		}
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)
		if err := c.CancelJob(cmd.Context(), args[0]); err != nil {
			return err
		}
		fmt.Println("Job cancelled.")
		return nil
	},
}

var (
	jobOpsAll   bool
	jobOpsPage  int
	jobOpsLimit int
)

var jobsOpsCmd = &cobra.Command{
	Use:   "operations <job-id>",
	Short: "List operations within a job",
	Long: `List the operations of a job, one per target entity, with their execution steps.

Results are paged. Use --all to walk every page, which is what you want for a
job over a large fleet; otherwise you get the first page only.

Examples:
  og jobs operations <job-id>
  og jobs operations <job-id> --all
  og jobs operations <job-id> --page 2 --limit 100 --output json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)

		if jobOpsAll && jobOpsPage > 0 {
			return fmt.Errorf("--all walks every page; it cannot be combined with --page")
		}

		var ops []opengate.Operation
		if jobOpsAll {
			for op, err := range c.GetJobOperationsAll(cmd.Context(), args[0]) {
				if err != nil {
					return err
				}
				ops = append(ops, op)
			}
		} else {
			page := -1 // let the platform serve its first page
			if jobOpsPage > 0 {
				page = jobOpsPage
			}
			resp, err := c.GetJobOperationsPage(cmd.Context(), args[0], page, jobOpsLimit)
			if err != nil {
				return err
			}
			ops = resp.Operations
		}

		return printOperations(ops)
	},
}

// --- operations history ---

var (
	opsHistoryJob    string
	opsHistoryWhere  []string
	opsHistoryFilter string
	opsHistoryLimit  int
	opsHistoryPage   int
	opsHistoryAll    bool
)

var jobsHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Search closed operations across jobs (with their execution steps)",
	Long: `Search the operation history: closed operations and their per-step results.

Unlike 'jobs operations', which lists one job by id, this takes a filter — so it
can pull the outcome of many operations in one walk. --job is a shortcut for the
common case of reading back a single job's results.

Results are paged; use --all to walk every page.

Filter fields are UNPREFIXED and the set is narrow (verified live): jobId, entityId,
operationId, resourceType and operationName (alias operation.name). Anything else —
including name, status, result, user, date and any "operations." prefix — is rejected
with HTTP 400 "Field in filter unknown".

There is NO server-side filter for status or result: fetch with --all and filter the
output yourself (e.g. jq) when you need "which entities failed".

Examples:
  og jobs history --job <job-id> --all
  og jobs history -w "operationName eq DIAGNOSIS" --limit 100
  og jobs history -w "entityId eq dev-1" --output json
  og jobs history --filter '{"filter":{"and":[{"eq":{"jobId":"<id>"}}]}}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)

		if opsHistoryAll && opsHistoryPage > 0 {
			return fmt.Errorf("--all walks every page; it cannot be combined with --page")
		}
		if opsHistoryJob != "" && (len(opsHistoryWhere) > 0 || opsHistoryFilter != "") {
			return fmt.Errorf("--job is a shortcut for a jobId filter; do not combine it with -w or --filter")
		}

		var filter json.RawMessage
		switch {
		case opsHistoryJob != "":
			filter = opengate.JobIDFilter(opsHistoryJob)
			if opsHistoryLimit > 0 || opsHistoryPage > 0 {
				page := opsHistoryPage
				if page < 1 {
					page = 1
				}
				size := opsHistoryLimit
				if size <= 0 {
					size = opengate.DefaultPageSize
				}
				if filter, err = opengate.SetFilterPage(filter, page, size); err != nil {
					return err
				}
			}
		default:
			filter, err = buildSearchFilterPaged(opsHistoryWhere, opsHistoryLimit, opsHistoryPage, nil, opsHistoryFilter)
			if err != nil {
				return err
			}
		}

		var ops []opengate.Operation
		if opsHistoryAll {
			for op, err := range c.SearchOperationsHistoryAll(cmd.Context(), filter) {
				if err != nil {
					return err
				}
				ops = append(ops, op)
			}
		} else {
			resp, err := c.SearchOperationsHistory(cmd.Context(), filter)
			if err != nil {
				return err
			}
			ops = resp.Operations
		}

		return printOperations(ops)
	},
}

// printOperations renders operations as JSON or as a table with a compact
// per-step summary, which is the column that actually answers "what failed".
func printOperations(ops []opengate.Operation) error {
	return output.Print(outFmt, ops,
		[]string{"Entity", "Operation", "Status", "Result", "Started", "Steps"},
		func(data any) [][]string {
			items := data.([]opengate.Operation)
			rows := make([][]string, len(items))
			for i, op := range items {
				started := ""
				if op.Execution != nil {
					started = op.Execution.StartedDate
				}
				rows[i] = []string{op.EntityID, op.Name, op.Status, op.Result, started, stepsSummary(op)}
			}
			return rows
		},
	)
}

// stepsSummary compacts the steps into "NAME=RESULT" pairs, abbreviating the
// result so a four-step diagnosis still fits a terminal column.
func stepsSummary(op opengate.Operation) string {
	if len(op.Steps) == 0 {
		return ""
	}
	parts := make([]string, 0, len(op.Steps))
	for _, s := range op.Steps {
		short := s.Result
		switch s.Result {
		case opengate.StepResultSuccessful:
			short = "OK"
		case opengate.StepResultError:
			short = "ERR"
		case opengate.StepResultSkipped:
			short = "SKIP"
		case opengate.StepResultNotExecuted:
			short = "NOTRUN"
		}
		parts = append(parts, s.Name+"="+short)
	}
	return strings.Join(parts, " ")
}

// --- tasks ---

var tasksCmd = &cobra.Command{
	Use:     "tasks",
	Aliases: []string{"task"},
	Short:   "Manage OpenGate operation tasks",
}

var tasksSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search tasks",
	RunE:  runTasksSearch,
}

var (
	taskSearchFilter string
	taskSearchWhere  []string
	taskSearchLimit  int
)

func runTasksSearch(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}
	c := opengate.New(p.Host, p.Token)

	filter, err := buildSearchFilter(taskSearchWhere, taskSearchLimit, nil, taskSearchFilter)
	if err != nil {
		return err
	}

	resp, err := c.SearchTasks(cmd.Context(), filter)
	if err != nil {
		return err
	}

	if outFmt == output.FormatJSON {
		return output.PrintJSON(os.Stdout, resp.Tasks)
	}

	rows := make([][]string, len(resp.Tasks))
	for i, raw := range resp.Tasks {
		var m map[string]any
		json.Unmarshal(raw, &m)
		id, _ := m["id"].(string)
		name, _ := m["name"].(string)
		state, _ := m["state"].(string)
		rows[i] = []string{id, name, state}
	}

	output.PrintTable(os.Stdout, []string{"ID", "Name", "State"}, rows)
	return nil
}

var tasksGetCmd = &cobra.Command{
	Use:   "get <task-id>",
	Short: "Get a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)

		data, err := c.GetTask(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		return output.PrintJSON(os.Stdout, json.RawMessage(data))
	},
}

var tasksCreateCmd = &cobra.Command{
	Use:   "create -f <file.json>",
	Short: "Create a new operation task",
	RunE:  runTasksCreate,
}

var taskCreateFile string

func runTasksCreate(cmd *cobra.Command, args []string) error {
	p, err := activeProfile()
	if err != nil {
		return err
	}

	body, err := os.ReadFile(taskCreateFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	c := opengate.New(p.Host, p.Token)
	resp, err := c.CreateTask(cmd.Context(), body)
	if err != nil {
		return err
	}

	return output.PrintJSON(os.Stdout, json.RawMessage(resp))
}

var tasksCancelCmd = &cobra.Command{
	Use:   "cancel <task-id>",
	Short: "Cancel a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := confirmDestructive(fmt.Sprintf("cancel task %q", args[0])); err != nil {
			return err
		}
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)
		if err := c.CancelTask(cmd.Context(), args[0]); err != nil {
			return err
		}
		fmt.Println("Task cancelled.")
		return nil
	},
}

var tasksJobsCmd = &cobra.Command{
	Use:   "jobs <task-id>",
	Short: "List jobs within a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := opengate.New(p.Host, p.Token)

		resp, err := c.GetTaskJobs(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		return output.PrintJSON(os.Stdout, resp.Jobs)
	},
}

// --- init ---

func init() {
	jobsSearchCmd.Flags().StringArrayVarP(&jobSearchWhere, "where", "w", nil, `filter condition (repeatable)`)
	jobsSearchCmd.Flags().IntVar(&jobSearchLimit, "limit", 0, "max results")
	jobsSearchCmd.Flags().StringVar(&jobSearchFilter, "filter", "", "raw JSON filter")

	jobsCreateCmd.Flags().StringVarP(&jobCreateFile, "file", "f", "", "JSON file with job definition")
	jobsCreateCmd.MarkFlagRequired("file")

	jobsOpsCmd.Flags().BoolVar(&jobOpsAll, "all", false, "fetch every page instead of just the first")
	jobsOpsCmd.Flags().IntVar(&jobOpsPage, "page", 0, "page number to fetch (default: the platform's first page)")
	jobsOpsCmd.Flags().IntVar(&jobOpsLimit, "limit", 0, "page size (max 1000 on this endpoint)")

	jobsHistoryCmd.Flags().StringVar(&opsHistoryJob, "job", "", "shortcut: read back the operations of this job id")
	jobsHistoryCmd.Flags().StringArrayVarP(&opsHistoryWhere, "where", "w", nil, `filter condition (repeatable)`)
	jobsHistoryCmd.Flags().StringVar(&opsHistoryFilter, "filter", "", "raw JSON filter")
	jobsHistoryCmd.Flags().IntVar(&opsHistoryLimit, "limit", 0, "page size (max 2000)")
	jobsHistoryCmd.Flags().IntVar(&opsHistoryPage, "page", 0, "page number to fetch, counting from 1")
	jobsHistoryCmd.Flags().BoolVar(&opsHistoryAll, "all", false, "fetch every page instead of just the first")

	jobsCmd.AddCommand(jobsSearchCmd)
	jobsCmd.AddCommand(jobsGetCmd)
	jobsCmd.AddCommand(jobsCreateCmd)
	jobsCmd.AddCommand(jobsCancelCmd)
	jobsCmd.AddCommand(jobsOpsCmd)
	jobsCmd.AddCommand(jobsHistoryCmd)
	rootCmd.AddCommand(jobsCmd)

	tasksSearchCmd.Flags().StringArrayVarP(&taskSearchWhere, "where", "w", nil, `filter condition (repeatable)`)
	tasksSearchCmd.Flags().IntVar(&taskSearchLimit, "limit", 0, "max results")
	tasksSearchCmd.Flags().StringVar(&taskSearchFilter, "filter", "", "raw JSON filter")

	tasksCreateCmd.Flags().StringVarP(&taskCreateFile, "file", "f", "", "JSON file with task definition")
	tasksCreateCmd.MarkFlagRequired("file")

	tasksCmd.AddCommand(tasksSearchCmd)
	tasksCmd.AddCommand(tasksGetCmd)
	tasksCmd.AddCommand(tasksCreateCmd)
	tasksCmd.AddCommand(tasksCancelCmd)
	tasksCmd.AddCommand(tasksJobsCmd)
	rootCmd.AddCommand(tasksCmd)
}
