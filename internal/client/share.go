package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	shareWorkspacePath = "/api/workspaces/%s/share"
	shareDashboardPath = "/api/dashboards/%s/share"
)

// ShareRequest is the body of the workspace/dashboard share endpoints.
// The PUT REPLACES both lists; empty lists unshare. (Captured live from the
// web UI — the wapi spec documents the endpoints without a body schema.)
type ShareRequest struct {
	Users   []string `json:"users"`
	Domains []string `json:"domains"`
}

// ShareWorkspace shares a workspace with the given users/domains. This is the
// ONLY mechanism that grants visibility to other users — setting users[] via
// the regular workspace PUT does not.
func (c *Client) ShareWorkspace(id string, users, domains []string) (json.RawMessage, error) {
	return c.share(fmt.Sprintf(shareWorkspacePath, id), "workspace", users, domains)
}

// ShareDashboard shares a single dashboard with the given users/domains.
func (c *Client) ShareDashboard(id string, users, domains []string) (json.RawMessage, error) {
	return c.share(fmt.Sprintf(shareDashboardPath, id), "dashboard", users, domains)
}

func (c *Client) share(path, kind string, users, domains []string) (json.RawMessage, error) {
	if users == nil {
		users = []string{}
	}
	if domains == nil {
		domains = []string{}
	}
	body, err := json.Marshal(ShareRequest{Users: users, Domains: domains})
	if err != nil {
		return nil, fmt.Errorf("marshaling share request: %w", err)
	}

	data, statusCode, err := c.WebPut(path, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("share %s: %w", kind, err)
	}
	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}
	return data, nil
}
