package mcp

import "testing"

func TestResolveToolsetsDefaultsToObserve(t *testing.T) {
	active, err := resolveToolsets(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// observe = curated read core + meta.
	for _, ts := range []toolset{tsDevices, tsAlarms, tsDatamodels, tsTimeseries, tsDatasets, tsJobs, tsMeta, tsLogin} {
		if !active[ts] {
			t.Errorf("observe should include %s", ts)
		}
	}
	// observe must NOT include any mutation or iot.
	for _, ts := range []toolset{tsDevicesWrite, tsAlarmsOps, tsIoT, tsRulesWrite, tsProvisionBulk} {
		if active[ts] {
			t.Errorf("observe must not include %s", ts)
		}
	}
}

func TestResolveToolsetsAll(t *testing.T) {
	active, err := resolveToolsets([]string{"all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, ts := range append(append([]toolset{}, readToolsets...), writeToolsets...) {
		if !active[ts] {
			t.Errorf("all should include %s", ts)
		}
	}
	if !active[tsIoT] {
		t.Error("all should include iot")
	}
}

func TestResolveToolsetsReadonlyExcludesWrites(t *testing.T) {
	active, err := resolveToolsets([]string{"readonly"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !active[tsRules] {
		t.Error("readonly should include the rules read group")
	}
	for _, ts := range writeToolsets {
		if active[ts] {
			t.Errorf("readonly must not include write/ops group %s", ts)
		}
	}
}

func TestResolveToolsetsIndividualGroups(t *testing.T) {
	active, err := resolveToolsets([]string{"devices", "alarms-ops"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !active[tsDevices] || !active[tsAlarmsOps] {
		t.Error("expected devices and alarms-ops to be active")
	}
	if active[tsAlarms] {
		t.Error("alarms read group was not requested and must not be active")
	}
	// meta is always present so the client can discover more.
	if !active[tsMeta] {
		t.Error("meta should always be active")
	}
}

func TestResolveToolsetsUnknownIsError(t *testing.T) {
	if _, err := resolveToolsets([]string{"bogus"}); err == nil {
		t.Fatal("expected error for unknown toolset")
	}
}

func TestResolveOptionsLeanIgnoresToolsets(t *testing.T) {
	opts, err := ResolveOptions(Options{Host: "h"}, true, []string{"all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Lean {
		t.Error("expected Lean to be true")
	}
	if opts.Toolsets != nil {
		t.Error("lean mode should not resolve a toolset map")
	}
	if opts.Host != "h" {
		t.Error("base fields should be preserved")
	}
}

func TestResolveOptionsToolsetMode(t *testing.T) {
	opts, err := ResolveOptions(Options{}, false, []string{"observe"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Lean {
		t.Error("expected Lean to be false")
	}
	if !opts.Toolsets[tsDevices] {
		t.Error("observe should activate the devices group")
	}
}
