package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerLeanTools exposes the exec-over-CLI surface. Instead of ~88 named
// tools — each loading its full JSON schema into the model's context on every
// request (~20-30k tokens) — the model drives the complete `og` CLI through two
// tools and discovers the surface on demand with og_help. This is the default
// for the stdio transport (a shell-capable, trusted, single-user agent).
func registerLeanTools(r *registrar) {
	r.tool(tsMeta, ogExecTool(), ogExecHandler(r))
	r.tool(tsMeta, ogHelpTool(), ogHelpHandler(r))
}

func ogExecTool() mcp.Tool {
	return mcp.NewTool("og_exec",
		mcp.WithDescription(`Run an og CLI subcommand and return its output. og is the OpenGate IoT platform CLI; this is its complete surface (devices, datamodels, alarms, timeseries, datasets, jobs, tasks, rules, connectors, provision, workspaces, dashboards, iot).

Pass the subcommand WITHOUT the leading "og". Quote arguments with spaces.
Top-level commands are plural (aliases in parens): devices (dev), datamodels (dm), alarms, timeseries (ts), datasets (ds), jobs, tasks, rules, connectors, provision, workspaces, dashboards, iot. The SINGULAR forms (device, datamodel, alarm) are NOT valid.
Examples:
  command: "dev search -w \"wt gt 20\" --view summary --output json"
  command: "dev get <device-id> --output json"
  command: "dm get <datamodel-id> --output json"
  command: "alarms summary"
Use og_help first to discover subcommands and flags. Prefer --output json for parseable results and project fields (--select / --view) to keep results small; for listings use "--view summary" rather than fetching full documents.

If a command fails, READ the error: it usually names the correct command ("did you mean ..."), the valid operators, or the missing field. Fix it and retry once; never repeat the exact same failing command. When a tool result already contains the answer, reply to the user with it — do not keep calling tools.

DESTRUCTIVE subcommands (delete, cancel) refuse to run without a terminal unless you pass --yes. Only add --yes when the user has explicitly consented to the destruction.`),
		mcp.WithString("command",
			mcp.Description("The og subcommand line, without the leading \"og\". E.g. \"dev search --view summary --output json\"."),
			mcp.Required(),
		),
	)
}

func ogExecHandler(r *registrar) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cmdline, _ := request.GetArguments()["command"].(string)
		if strings.TrimSpace(cmdline) == "" {
			return mcp.NewToolResultError(`command is required, e.g. "dev search -w 'wt gt 20' --output json"`), nil
		}
		tokens, err := tokenize(cmdline)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("parsing command: %v", err)), nil
		}
		// Tolerate a stray leading "og".
		if len(tokens) > 0 && tokens[0] == "og" {
			tokens = tokens[1:]
		}
		if len(tokens) == 0 {
			return mcp.NewToolResultError("empty command"), nil
		}
		return runOG(ctx, r, tokens)
	}
}

func ogHelpTool() mcp.Tool {
	return mcp.NewTool("og_help",
		mcp.WithDescription("Show the help for an og subcommand (runs `og <path> --help`). Use it to discover the available subcommands, arguments and flags before calling og_exec. Pass an empty path for the top-level help."),
		mcp.WithString("path",
			mcp.Description("Subcommand path without \"og\", e.g. \"devices\" or \"dev search\". Empty for top-level help."),
		),
	)
}

func ogHelpHandler(r *registrar) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, _ := request.GetArguments()["path"].(string)
		tokens, err := tokenize(path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("parsing path: %v", err)), nil
		}
		if len(tokens) > 0 && tokens[0] == "og" {
			tokens = tokens[1:]
		}
		tokens = append(tokens, "--help")
		return runOG(ctx, r, tokens)
	}
}

// runOG executes the running og binary with the given args, injecting the
// request's resolved credentials as environment variables so the child process
// authenticates with the same identity as the MCP server (essential in
// multi-tenant mode, where the credentials come per-request from headers and are
// not on disk). The combined stdout+stderr is returned to the model.
func runOG(ctx context.Context, r *registrar, args []string) (*mcp.CallToolResult, error) {
	creds, err := r.p.credsErr(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	self, err := os.Executable()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("locating og binary: %v", err)), nil
	}

	env := os.Environ()
	if r.p.host != "" {
		env = append(env, "OG_HOST="+r.p.host)
	}
	if creds.token != "" {
		env = append(env, "OG_TOKEN="+creds.token)
	}
	if creds.webToken != "" {
		env = append(env, "OG_WEB_TOKEN="+creds.webToken)
	}

	cmd := exec.CommandContext(ctx, self, args...)
	cmd.Env = env
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	runErr := cmd.Run()

	combined := out.String()
	if errb.Len() > 0 {
		if combined != "" && !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		combined += errb.String()
	}
	if runErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("og exited with error: %v\n%s", runErr, combined)), nil
	}
	if strings.TrimSpace(combined) == "" {
		combined = "(no output; command succeeded)"
	}
	return mcp.NewToolResultText(combined), nil
}

// tokenize splits a command line into arguments, honouring single and double
// quotes so arguments with spaces (e.g. -w "wt gt 20") survive intact. It is a
// minimal shell-like splitter — no escaping or variable expansion.
func tokenize(s string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	inToken := false
	var quote rune

	for _, ru := range s {
		switch {
		case quote != 0:
			if ru == quote {
				quote = 0
			} else {
				cur.WriteRune(ru)
			}
		case ru == '\'' || ru == '"':
			quote = ru
			inToken = true
		case ru == ' ' || ru == '\t' || ru == '\n' || ru == '\r':
			if inToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
		default:
			cur.WriteRune(ru)
			inToken = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote (%c)", quote)
	}
	if inToken {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}
