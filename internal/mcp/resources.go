package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerResources(s *server.MCPServer, c *client.Client) {
	// Static resource: query syntax reference
	s.AddResource(
		mcp.NewResource(
			"opengate://query-syntax",
			"OpenGate query syntax reference",
			mcp.WithMIMEType("text/plain"),
		),
		handleQuerySyntaxResource,
	)

	// Dynamic resource template: datamodel fields per organization
	s.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"opengate://organizations/{org}/datamodel-fields",
			"Available datastream fields for an organization's datamodels",
		),
		datamodelFieldsHandler(c),
	)
}

func handleQuerySyntaxResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "opengate://query-syntax",
			MIMEType: "text/plain",
			Text: `OpenGate Query Syntax
=====================

Format: "field operator value"
Multiple: "field1 op value1 AND field2 op value2"

Operators:
  eq      — equals (exact match)
  neq     — not equals
  like    — contains / partial match
  gt      — greater than
  lt      — less than
  gte     — greater than or equal
  lte     — less than or equal
  in      — one of (comma-separated values)
  exists  — field exists (no value needed)

Device fields:
  provision.device.identifier
  provision.device.name
  provision.device.administrativeState   (ACTIVE, TESTING, BANNED)
  provision.device.operationalStatus     (NORMAL, ALARM, DOWN)
  provision.administration.organization
  provision.administration.channel

Datamodel fields:
  datamodels.identifier
  datamodels.name
  datamodels.organizationName
  datamodels.version

Custom datastream fields depend on the organization's datamodels.
Use opengate://organizations/{org}/datamodel-fields to discover them.
`,
		},
	}, nil
}

func datamodelFieldsHandler(c *client.Client) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract org from URI: opengate://organizations/{org}/datamodel-fields
		uri := request.Params.URI
		parts := strings.Split(uri, "/")
		var orgName string
		for i, p := range parts {
			if p == "organizations" && i+1 < len(parts) {
				orgName = parts[i+1]
				break
			}
		}
		if orgName == "" {
			return nil, fmt.Errorf("organization not found in URI: %s", uri)
		}

		resp, err := c.SearchDatamodels(nil)
		if err != nil {
			return nil, fmt.Errorf("fetching datamodels: %w", err)
		}

		var b strings.Builder
		b.WriteString(fmt.Sprintf("Available datastream fields for organization: %s\n", orgName))
		b.WriteString("==========================================================\n\n")

		for _, dm := range resp.Datamodels {
			if dm.OrganizationName != orgName {
				continue
			}
			b.WriteString(fmt.Sprintf("Datamodel: %s (v%s) — %s\n", dm.Identifier, dm.Version, dm.Name))
			for _, cat := range dm.Categories {
				for _, ds := range cat.Datastreams {
					b.WriteString(fmt.Sprintf("  %-50s  %s\n", ds.Identifier, ds.Name))
				}
			}
			b.WriteString("\n")
		}

		// Need full detail for custom datamodels — fetch each one
		for _, dm := range resp.Datamodels {
			if dm.OrganizationName != orgName {
				continue
			}
			if len(dm.Categories) == 0 {
				detail, err := c.GetDatamodel(orgName, dm.Identifier)
				if err != nil {
					continue
				}
				if len(detail.Categories) > 0 {
					b.WriteString(fmt.Sprintf("Datamodel: %s (v%s) — %s\n", detail.Identifier, detail.Version, detail.Name))
					for _, cat := range detail.Categories {
						for _, ds := range cat.Datastreams {
							b.WriteString(fmt.Sprintf("  %-50s  %s\n", ds.Identifier, ds.Name))
						}
					}
					b.WriteString("\n")
				}
			}
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "text/plain",
				Text:     b.String(),
			},
		}, nil
	}
}
