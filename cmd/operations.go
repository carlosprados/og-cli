package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/carlosprados/og-cli/internal/output"
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
	c := client.New(p.Host, p.Token)

	filter, err := buildSearchFilter(jobSearchWhere, jobSearchLimit, nil, jobSearchFilter)
	if err != nil {
		return err
	}

	resp, err := c.SearchJobs(filter)
	if err != nil {
		return err
	}

	if outFmt == output.FormatJSON {
		return output.PrintJSON(os.Stdout, resp.Jobs)
	}

	// Jobs are complex JSON — extract key fields
	type jobSummary struct {
		ID     string `json:"id"`
		Name   string `json:"request.name"`
		Status string `json:"report.summary.status"`
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
		c := client.New(p.Host, p.Token)

		data, err := c.GetJob(args[0])
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

	c := client.New(p.Host, p.Token)
	resp, err := c.CreateJob(body)
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
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := client.New(p.Host, p.Token)
		if err := c.CancelJob(args[0]); err != nil {
			return err
		}
		fmt.Println("Job cancelled.")
		return nil
	},
}

var jobsOpsCmd = &cobra.Command{
	Use:   "operations <job-id>",
	Short: "List operations within a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := client.New(p.Host, p.Token)

		resp, err := c.GetJobOperations(args[0])
		if err != nil {
			return err
		}

		return output.PrintJSON(os.Stdout, resp.Operations)
	},
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
	c := client.New(p.Host, p.Token)

	filter, err := buildSearchFilter(taskSearchWhere, taskSearchLimit, nil, taskSearchFilter)
	if err != nil {
		return err
	}

	resp, err := c.SearchTasks(filter)
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
		c := client.New(p.Host, p.Token)

		data, err := c.GetTask(args[0])
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

	c := client.New(p.Host, p.Token)
	resp, err := c.CreateTask(body)
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
		p, err := activeProfile()
		if err != nil {
			return err
		}
		c := client.New(p.Host, p.Token)
		if err := c.CancelTask(args[0]); err != nil {
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
		c := client.New(p.Host, p.Token)

		resp, err := c.GetTaskJobs(args[0])
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

	jobsCmd.AddCommand(jobsSearchCmd)
	jobsCmd.AddCommand(jobsGetCmd)
	jobsCmd.AddCommand(jobsCreateCmd)
	jobsCmd.AddCommand(jobsCancelCmd)
	jobsCmd.AddCommand(jobsOpsCmd)
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
