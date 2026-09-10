package opengate

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The platform answers a lookup that matches nothing with 204 and an empty
// body. Verified live 2026-09-10 against datamodels, workspaces, dashboards
// and devices — all four, plus every family's --raw sibling, used to report
// that as either a JSON parse error or as nothing at all.
func emptyBodyServer(t *testing.T) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "token")
	c.WebToken = "token"
	return c
}

func TestGettersReportAMissingArtifact(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		call func(*Client) error
		kind string
		id   string
	}{
		"GetDatamodel": {func(c *Client) error {
			_, err := c.GetDatamodel(ctx, "org", "NoExiste")
			return err
		}, "datamodel", "NoExiste"},
		"GetDatamodelRaw": {func(c *Client) error {
			_, err := c.GetDatamodelRaw(ctx, "org", "NoExiste")
			return err
		}, "datamodel", "NoExiste"},
		"GetWorkspace": {func(c *Client) error {
			_, err := c.GetWorkspace(ctx, "Operations", false)
			return err
		}, "workspace", "Operations"},
		"GetWorkspaceRaw": {func(c *Client) error {
			_, err := c.GetWorkspaceRaw(ctx, "Operations", false)
			return err
		}, "workspace", "Operations"},
		"GetDashboard": {func(c *Client) error {
			_, err := c.GetDashboard(ctx, "Overview")
			return err
		}, "dashboard", "Overview"},
		"GetDevice": {func(c *Client) error {
			_, err := c.GetDevice(ctx, "org", "dev-1")
			return err
		}, "device", "dev-1"},
		"GetTimeSeries": {func(c *Client) error {
			_, err := c.GetTimeSeries(ctx, "org", "ts-1")
			return err
		}, "time series", "ts-1"},
		"GetTimeSeriesRaw": {func(c *Client) error {
			_, err := c.GetTimeSeriesRaw(ctx, "org", "ts-1")
			return err
		}, "time series", "ts-1"},
		"GetDataset": {func(c *Client) error {
			_, err := c.GetDataset(ctx, "org", "ds-1")
			return err
		}, "dataset", "ds-1"},
		"GetRule": {func(c *Client) error {
			_, err := c.GetRule(ctx, "org", "ch", "r-1")
			return err
		}, "rule", "r-1"},
		"GetConnectorFunction": {func(c *Client) error {
			_, err := c.GetConnectorFunction(ctx, "org", "ch", "cf-1")
			return err
		}, "connector function", "cf-1"},
		"GetProvisionProcessor": {func(c *Client) error {
			_, err := c.GetProvisionProcessor(ctx, "org", "pp-1")
			return err
		}, "provision processor", "pp-1"},
	}

	for name, tc := range cases {
		c := emptyBodyServer(t)
		err := tc.call(c)
		if err == nil {
			t.Errorf("%s: want an error on an empty body, got none", name)
			continue
		}
		if !IsNotFound(err) {
			t.Errorf("%s: error is %v, want a NotFoundError", name, err)
			continue
		}
		var nf *NotFoundError
		_ = errors.As(err, &nf)
		if nf.Kind != tc.kind || nf.Identifier != tc.id {
			t.Errorf("%s: got kind=%q id=%q, want kind=%q id=%q", name, nf.Kind, nf.Identifier, tc.kind, tc.id)
		}
	}
}

// A list answering 204 means "none yet". That is an answer, and turning it
// into an error would break every empty tenant.
func TestListsTreatAnEmptyBodyAsEmpty(t *testing.T) {
	ctx := context.Background()
	c := emptyBodyServer(t)

	if resp, err := c.ListTimeSeries(ctx, "org"); err != nil {
		t.Errorf("ListTimeSeries: %v", err)
	} else if len(resp.Timeseries) != 0 {
		t.Errorf("ListTimeSeries returned %d entries, want 0", len(resp.Timeseries))
	}

	if raw, err := c.ListTimeSeriesRaw(ctx, "org"); err != nil {
		t.Errorf("ListTimeSeriesRaw: %v", err)
	} else if string(raw) != "{}" {
		t.Errorf("ListTimeSeriesRaw = %s, want {}", raw)
	}
}

// A body that is actually there must pass through untouched.
func TestGetterKeepsAPresentArtifact(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"identifier":"ts-1","name":"Consumo"}`))
	}))
	defer srv.Close()

	ts, err := New(srv.URL, "token").GetTimeSeries(context.Background(), "org", "ts-1")
	if err != nil {
		t.Fatalf("GetTimeSeries: %v", err)
	}
	if ts.Name != "Consumo" {
		t.Errorf("name = %q, want Consumo", ts.Name)
	}
}
