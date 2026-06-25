package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// toolset is a named group of MCP tools (plus their associated prompts and
// resources). Toolsets let a client load only the surface it needs instead of
// every tool at once — the full set costs ~20-30k tokens of context on every
// request, whether used or not. The per-transport defaults live in cmd/mcp.go:
// stdio defaults to the lean exec-over-CLI mode, HTTP to the curated "observe"
// alias. See docs/mcp-integration.md.
type toolset string

const (
	// login is single-tenant only (it returns a JWT in its result text) and is
	// dropped in multi-tenant mode regardless of the active set.
	tsLogin toolset = "login"

	// Read toolsets — non-mutating search/get/list/data/summary/logs/catalog.
	tsDevices    toolset = "devices"
	tsDatamodels toolset = "datamodels"
	tsAlarms     toolset = "alarms"
	tsTimeseries toolset = "timeseries"
	tsDatasets   toolset = "datasets"
	tsJobs       toolset = "jobs"
	tsRules      toolset = "rules"
	tsConnectors toolset = "connectors"
	tsProvision  toolset = "provision"
	tsOptypes    toolset = "optypes"
	tsWorkspaces toolset = "workspaces"
	tsDashboards toolset = "dashboards"
	tsTasks      toolset = "tasks"

	// Write/ops toolsets — mutations, lifecycle and high-impact actions.
	tsDevicesWrite    toolset = "devices-write"
	tsDatamodelsWrite toolset = "datamodels-write"
	tsAlarmsOps       toolset = "alarms-ops"
	tsTimeseriesWrite toolset = "timeseries-write"
	tsDatasetsWrite   toolset = "datasets-write"
	tsJobsRun         toolset = "jobs-run"
	tsRulesWrite      toolset = "rules-write"
	tsRulesOps        toolset = "rules-ops"
	tsConnectorsWrite toolset = "connectors-write"
	tsConnectorsOps   toolset = "connectors-ops"
	tsProvisionWrite  toolset = "provision-write"
	tsProvisionBulk   toolset = "provision-bulk"
	tsOptypesWrite    toolset = "optypes-write"
	tsWorkspacesWrite toolset = "workspaces-write"
	tsWorkspacesShare toolset = "workspaces-share"
	tsDashboardsWrite toolset = "dashboards-write"
	tsDashboardsShare toolset = "dashboards-share"
	tsTasksWrite      toolset = "tasks-write"

	// iot is inject/MQTT only (no read tools); never part of a read default.
	tsIoT toolset = "iot"

	// meta is the og_toolsets discovery tool; always on in non-lean modes so a
	// client can see what else exists and request it.
	tsMeta toolset = "meta"
)

// readToolsets are the non-mutating groups (the "readonly" alias).
var readToolsets = []toolset{
	tsDevices, tsDatamodels, tsAlarms, tsTimeseries, tsDatasets, tsJobs,
	tsRules, tsConnectors, tsProvision, tsOptypes, tsWorkspaces, tsDashboards, tsTasks,
}

// writeToolsets are the mutating / high-impact groups.
var writeToolsets = []toolset{
	tsDevicesWrite, tsDatamodelsWrite, tsAlarmsOps, tsTimeseriesWrite, tsDatasetsWrite,
	tsJobsRun, tsRulesWrite, tsRulesOps, tsConnectorsWrite, tsConnectorsOps,
	tsProvisionWrite, tsProvisionBulk, tsOptypesWrite, tsWorkspacesWrite, tsWorkspacesShare,
	tsDashboardsWrite, tsDashboardsShare, tsTasksWrite, tsIoT,
}

// observeToolsets is the curated, low-token default for HTTP: the read surface
// a monitoring/Q&A agent needs, and nothing that mutates. The client opts into
// more via --toolsets (or the og_toolsets discovery tool tells it what exists).
var observeToolsets = []toolset{
	tsDevices, tsAlarms, tsDatamodels, tsTimeseries, tsDatasets, tsJobs,
}

// toolsetDescriptions documents each group for the og_toolsets discovery tool.
var toolsetDescriptions = map[toolset]string{
	tsLogin:           "authenticate and obtain a JWT (single-tenant only)",
	tsDevices:         "search/get devices",
	tsDevicesWrite:    "create/update/delete devices",
	tsDatamodels:      "search/get datamodels",
	tsDatamodelsWrite: "create/update/delete datamodels",
	tsAlarms:          "alarm summary + search",
	tsAlarmsOps:       "attend/close alarms",
	tsTimeseries:      "read timeseries (data/get/list)",
	tsTimeseriesWrite: "create/update/delete timeseries",
	tsDatasets:        "read datasets (data/get/list)",
	tsDatasetsWrite:   "create/update/delete datasets",
	tsJobs:            "search/get operation jobs + operations",
	tsJobsRun:         "create/cancel operation jobs",
	tsRules:           "search/get rules + logs + catalog",
	tsRulesWrite:      "create/update/delete rules",
	tsRulesOps:        "activate/deactivate rules",
	tsConnectors:      "list/get connectors + logs + catalog",
	tsConnectorsWrite: "create/update/delete connectors",
	tsConnectorsOps:   "set connector status",
	tsProvision:       "list/get provision functions + bulk status",
	tsProvisionWrite:  "create/update/delete provision functions",
	tsProvisionBulk:   "run provision bulk + plan",
	tsOptypes:         "search/get operation types + catalog",
	tsOptypesWrite:    "create/update/delete operation types",
	tsWorkspaces:      "list/get workspaces + export",
	tsWorkspacesWrite: "create/update/delete + import workspaces",
	tsWorkspacesShare: "share workspaces",
	tsDashboards:      "list/get dashboards + export",
	tsDashboardsWrite: "create/update/delete + import dashboards",
	tsDashboardsShare: "share dashboards",
	tsTasks:           "search/get scheduled tasks",
	tsTasksWrite:      "create/cancel scheduled tasks",
	tsIoT:             "inject IoT data (HTTP/MQTT) + virtual device",
	tsMeta:            "discovery: list available toolsets",
}

// ResolveOptions fills the surface fields (Lean / Toolsets) of an Options value
// from the CLI intent. It is the seam cmd/ uses, so the unexported toolset type
// and resolution rules stay inside this package. toolsetNames accepts the
// observe/readonly/all aliases or individual group names; an unknown name is a
// hard error.
func ResolveOptions(base Options, lean bool, toolsetNames []string) (Options, error) {
	base.Lean = lean
	if lean {
		base.Toolsets = nil
		return base, nil
	}
	ts, err := resolveToolsets(toolsetNames)
	if err != nil {
		return Options{}, err
	}
	base.Toolsets = ts
	return base, nil
}

// resolveToolsets expands the requested names (aliases observe/readonly/all or
// individual toolset names) into the set of active toolsets. tsMeta is always
// included so a client is never blind to what else it could enable. Unknown
// names are rejected so a typo fails loudly instead of silently disabling work.
func resolveToolsets(names []string) (map[toolset]bool, error) {
	active := map[toolset]bool{tsMeta: true}
	add := func(ts ...toolset) {
		for _, t := range ts {
			active[t] = true
		}
	}

	if len(names) == 0 {
		names = []string{"observe"}
	}

	for _, raw := range names {
		name := strings.TrimSpace(strings.ToLower(raw))
		if name == "" {
			continue
		}
		switch name {
		case "observe":
			add(tsLogin)
			add(observeToolsets...)
		case "readonly", "read":
			add(tsLogin)
			add(readToolsets...)
		case "all":
			add(tsLogin)
			add(readToolsets...)
			add(writeToolsets...)
		default:
			ts := toolset(name)
			if _, ok := toolsetDescriptions[ts]; !ok {
				return nil, fmt.Errorf("unknown toolset %q (valid: observe, readonly, all, or one of %s)", raw, strings.Join(knownToolsetNames(), ", "))
			}
			add(ts)
		}
	}
	return active, nil
}

// knownToolsetNames returns the individual (non-alias) toolset names, sorted.
func knownToolsetNames() []string {
	out := make([]string, 0, len(toolsetDescriptions))
	for ts := range toolsetDescriptions {
		if ts == tsMeta {
			continue
		}
		out = append(out, string(ts))
	}
	sort.Strings(out)
	return out
}

// registrar wraps an MCPServer and registers tools, prompts and resources only
// when their toolset is in the active set. The 13 register*Tools functions and
// the prompt/resource registrars route every AddX call through here, so the
// active set alone decides the exposed surface — handlers are untouched.
type registrar struct {
	s      *server.MCPServer
	p      *provider
	host   string
	active map[toolset]bool
}

func (r *registrar) enabled(ts toolset) bool { return r.active[ts] }

func (r *registrar) tool(ts toolset, t mcp.Tool, h server.ToolHandlerFunc) {
	if r.enabled(ts) {
		r.s.AddTool(t, h)
	}
}

func (r *registrar) prompt(ts toolset, pr mcp.Prompt, h server.PromptHandlerFunc) {
	if r.enabled(ts) {
		r.s.AddPrompt(pr, h)
	}
}

func (r *registrar) resource(ts toolset, res mcp.Resource, h server.ResourceHandlerFunc) {
	if r.enabled(ts) {
		r.s.AddResource(res, h)
	}
}

func (r *registrar) resourceTemplate(ts toolset, tmpl mcp.ResourceTemplate, h server.ResourceTemplateHandlerFunc) {
	if r.enabled(ts) {
		r.s.AddResourceTemplate(tmpl, h)
	}
}

// registerToolsetsMeta adds the og_toolsets discovery tool, which reports the
// active and available toolsets so a client (or its LLM) can request more.
func registerToolsetsMeta(r *registrar) {
	r.tool(tsMeta, toolsetsMetaTool(), toolsetsMetaHandler(r.active))
}

func toolsetsMetaTool() mcp.Tool {
	return mcp.NewTool("og_toolsets",
		mcp.WithDescription("List the OpenGate MCP toolsets: which are active now and which can be enabled. To use more tools, restart the server with --toolsets <name,...> (aliases: observe, readonly, all). This server keeps the surface small to save context tokens."),
	)
}

func toolsetsMetaHandler(active map[toolset]bool) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var b strings.Builder
		b.WriteString("OpenGate MCP toolsets\n=====================\n\n")
		b.WriteString("ACTIVE (tools currently exposed):\n")
		for _, ts := range allToolsetsSorted() {
			if active[ts] {
				fmt.Fprintf(&b, "  %-20s — %s\n", ts, toolsetDescriptions[ts])
			}
		}
		b.WriteString("\nAVAILABLE (enable with --toolsets <name>):\n")
		for _, ts := range allToolsetsSorted() {
			if !active[ts] {
				fmt.Fprintf(&b, "  %-20s — %s\n", ts, toolsetDescriptions[ts])
			}
		}
		b.WriteString("\nAliases: observe (curated read core), readonly (all read), all (everything).\n")
		return mcp.NewToolResultText(b.String()), nil
	}
}

// allToolsetsSorted returns every toolset name sorted, for stable listing.
func allToolsetsSorted() []toolset {
	out := make([]toolset, 0, len(toolsetDescriptions))
	for ts := range toolsetDescriptions {
		out = append(out, ts)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
