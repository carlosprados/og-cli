package canon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosprados/og-cli/v2/internal/unwrap"
	"github.com/carlosprados/og-cli/v2/pkg/opengate"
)

// This is the acceptance criterion for canonicalization: for every artifact in
// the corpus, exploding it to a directory tree and imploding it back must
// produce the same artifact. Not the same bytes — the same artifact, which is
// what canon defines.
//
// The corpus is deliberately made of REAL API responses (pkg/opengate/testdata)
// and the real demo trees, not synthetic fixtures: the two data-loss bugs this
// guards against were both shapes nobody would have thought to write by hand.

// ── dashboards and workspaces: real GET responses ────────────────────────────

func TestRoundTripDashboardFixture(t *testing.T) {
	raw := readFixture(t, filepath.Join("..", "..", "pkg", "opengate", "testdata", "dashboard_get.json"))

	var d opengate.Dashboard
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}
	if len(d.Grid) == 0 {
		t.Fatal("fixture has no widgets — it would not exercise the widget cycle")
	}

	dir := filepath.Join(t.TempDir(), "dash")
	if err := unwrap.UnwrapDashboardFull(&d, nil, dir, nil); err != nil {
		t.Fatalf("unwrap: %v", err)
	}
	rebuilt, _, err := unwrap.WrapDashboard(dir, nil)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}

	before, err := json.Marshal(&d)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(rebuilt)
	if err != nil {
		t.Fatal(err)
	}

	opts := Options{Kind: unwrap.KindDashboard}
	assertSameArtifact(t, "dashboard_get.json", before, after, opts)
}

func TestRoundTripWorkspaceFixtures(t *testing.T) {
	// The nested family's cycle: workspace.json plus one directory per
	// dashboard, each with one directory per widget.
	//
	// Coverage limit, stated rather than skipped: a workspace GET embeds its
	// dashboards' layout but NOT their grids (the fixture's dashboard has an
	// empty grid — the platform requires a separate dashboard GET for it). So
	// this exercises the workspace→dashboard nesting, and
	// TestRoundTripDashboardFixture covers dashboard→widget with three real
	// widgets. No fixture covers all three levels at once because no single API
	// response contains them.
	for _, w := range loadWorkspaceFixtures(t) {
		t.Run(w.name, func(t *testing.T) {
			dir := t.TempDir()
			if _, err := unwrap.Unwrap(w.ws, dir); err != nil {
				t.Fatalf("unwrap workspace: %v", err)
			}
			for i, wd := range w.ws.Dashboards {
				// Carry every field across, the way a real dashboard GET does.
				// Copying a hand-picked few would make the comparison test the
				// test rather than the code.
				full := &opengate.Dashboard{}
				if wd.Dashboard != nil {
					body, err := json.Marshal(wd.Dashboard)
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(body, full); err != nil {
						t.Fatal(err)
					}
				}
				sub := filepath.Join(dir, fmt.Sprintf("%02d__%s", i, unwrap.Slugify(full.Title)))
				layout := wd
				if err := unwrap.UnwrapDashboardFull(full, &layout, sub, nil); err != nil {
					t.Fatalf("unwrap dashboard %d: %v", i, err)
				}
			}

			rebuilt, err := unwrap.Wrap(dir, nil)
			if err != nil {
				t.Fatalf("wrap: %v", err)
			}

			if len(rebuilt.Dashboards) != len(w.ws.Dashboards) {
				t.Fatalf("dashboards: got %d, want %d", len(rebuilt.Dashboards), len(w.ws.Dashboards))
			}

			before, err := json.Marshal(w.ws)
			if err != nil {
				t.Fatal(err)
			}
			after, err := json.Marshal(rebuilt)
			if err != nil {
				t.Fatal(err)
			}
			assertSameArtifact(t, w.name, before, after, Options{Kind: unwrap.KindWorkspace})
		})
	}
}

type namedWorkspace struct {
	name string
	ws   *opengate.Workspace
}

// loadWorkspaceFixtures reads every workspace in the corpus: the bare GET
// response and each entry of the export envelope, which is {"workspaces":[...]}.
func loadWorkspaceFixtures(t *testing.T) []namedWorkspace {
	t.Helper()
	base := filepath.Join("..", "..", "pkg", "opengate", "testdata")

	var out []namedWorkspace

	var single opengate.Workspace
	if err := json.Unmarshal(readFixture(t, filepath.Join(base, "workspace_get.json")), &single); err != nil {
		t.Fatalf("parsing workspace_get.json: %v", err)
	}
	out = append(out, namedWorkspace{"workspace_get.json", &single})

	var envelope struct {
		Workspaces []opengate.Workspace `json:"workspaces"`
	}
	if err := json.Unmarshal(readFixture(t, filepath.Join(base, "workspace_export.json")), &envelope); err != nil {
		t.Fatalf("parsing workspace_export.json: %v", err)
	}
	if len(envelope.Workspaces) == 0 {
		t.Fatal("workspace_export.json holds no workspaces — the fixture or the envelope shape changed")
	}
	for i := range envelope.Workspaces {
		ws := envelope.Workspaces[i]
		out = append(out, namedWorkspace{
			fmt.Sprintf("workspace_export.json[%d] %s", i, ws.Name), &ws,
		})
	}
	return out
}

// ── flat families: every descriptor, over the demo trees and real shapes ─────

func TestRoundTripFlatFamilies(t *testing.T) {
	cases := []struct {
		name    string
		d       unwrap.Descriptor
		kind    unwrap.Kind
		payload string
	}{
		{
			name: "rule ADVANCED", d: unwrap.RuleDescriptor(), kind: unwrap.KindRule,
			payload: readDemoRule(t, "env-anomaly"),
		},
		{
			name: "rule EASY", d: unwrap.RuleDescriptor(), kind: unwrap.KindRule,
			payload: readDemoRule(t, "battery-low"),
		},
		{
			name: "connector function REQUEST", d: unwrap.ConnectorFunctionDescriptor(), kind: unwrap.KindConnectorFunction,
			payload: `{"identifier":"cf-1","name":"refreshInfo","type":"REQUEST","operationalStatus":"PRODUCTION",
			           "payloadType":"JSON","operationName":"REFRESH_INFO",
			           "northCriterias":[{"path":"provision.device.model._current.value.manufacturer","value":"Acme"}],
			           "javascript":"function buildRequest(operation) {\n  return {cmd: 'refresh'};\n}"}`,
		},
		{
			name: "connector function COLLECTION", d: unwrap.ConnectorFunctionDescriptor(), kind: unwrap.KindConnectorFunction,
			payload: `{"identifier":"cf-2","name":"collectWeather","type":"COLLECTION","operationalStatus":"TEST",
			           "southCriterias":[{"path":"/weather"}],
			           "javascript":"var c = ogCollection();\nreturn c;"}`,
		},
		{
			name: "provision function", d: unwrap.ProvisionFunctionDescriptor(), kind: unwrap.KindProvisionFunction,
			payload: `{"provisionProcessorId":"pp-1","name":"createUpdate",
			           "configurationParams":{"spreadsheet":{"sheetName":"PARA","headerRow":2,"resultColumnName":"ODM_Result"}},
			           "scriptProcessor":{"script":"function normalizeRawObject(o){return o;}\nfunction actionsPlanning(n){return [];}"}}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := json.RawMessage(c.payload)

			dir, err := c.d.Unwrap(before, t.TempDir(), &unwrap.Options{})
			if err != nil {
				t.Fatalf("unwrap: %v", err)
			}
			after, err := c.d.Wrap(dir, nil)
			if err != nil {
				t.Fatalf("wrap: %v", err)
			}
			assertSameArtifact(t, c.name, before, after, Options{Kind: c.kind})
		})
	}
}

// The cycle must also be idempotent: running it twice changes nothing further.
// A cycle that converges after two passes rather than one would make `diff`
// report a phantom change on the first save.
func TestRoundTripIsIdempotent(t *testing.T) {
	payload := json.RawMessage(readDemoRule(t, "env-anomaly"))
	d := unwrap.RuleDescriptor()
	opts := Options{Kind: unwrap.KindRule}

	first, err := cycle(t, d, payload)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cycle(t, d, first)
	if err != nil {
		t.Fatal(err)
	}
	assertSameArtifact(t, "second pass", first, second, opts)
}

// ── the shapes that used to lose data ───────────────────────────────────────

// Both of these round-tripped incorrectly before v2.1.0. They go through the
// real dashboard→widget cycle, on disk, because that is where they occur: the
// shapes need a family whose code fields nest at arbitrary depth, which the
// flat families' fixed keypaths never exercise. Routing them through a rule
// descriptor would make this test vacuous — nothing would be extracted, so
// nothing would be reinjected, and the bug would not be reached.
func TestRoundTripRegressionShapes(t *testing.T) {
	cases := map[string]string{
		"object keyed by number": `{"series":{"0":{"formatter":"function(v){return v;}"}}}`,
		"numeric key beside a real array": `{"m":{"2":{"code":"function(){return 1;}"}},
		                                    "a":[{"code":"function(){return 2;}"}]}`,
		"key containing the path separator": `{"a__b":{"code":"function(){return 1;}"},
		                                      "a":{"b":{"code":"function(){return 2;}"}}}`,
		"underscore run of three": `{"a___b":{"code":"function(){return 3;}"}}`,
		"percent in a key":        `{"a%b":{"code":"function(){return 4;}"}}`,
	}

	for label, config := range cases {
		t.Run(label, func(t *testing.T) {
			d := &opengate.Dashboard{
				ID: "d-1", Title: "T",
				Grid: []opengate.GridItem{{
					X: 0, Y: 0, W: 4, H: 2, I: "w1",
					Definition: &opengate.WidgetDefinition{
						Type: "TestWidget", Wid: "w1", Config: json.RawMessage(config),
					},
				}},
			}

			dir := filepath.Join(t.TempDir(), "dash")
			if err := unwrap.UnwrapDashboardFull(d, nil, dir, nil); err != nil {
				t.Fatalf("unwrap: %v", err)
			}
			rebuilt, _, err := unwrap.WrapDashboard(dir, nil)
			if err != nil {
				t.Fatalf("wrap: %v", err)
			}
			if len(rebuilt.Grid) != 1 || rebuilt.Grid[0].Definition == nil {
				t.Fatal("widget lost across the cycle")
			}

			assertSameArtifact(t, label,
				d.Grid[0].Definition.Config, rebuilt.Grid[0].Definition.Config,
				Options{Kind: unwrap.KindDashboard})
		})
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

func cycle(t *testing.T, d unwrap.Descriptor, payload json.RawMessage) (json.RawMessage, error) {
	t.Helper()
	dir, err := d.Unwrap(payload, t.TempDir(), &unwrap.Options{})
	if err != nil {
		return nil, err
	}
	return d.Wrap(dir, nil)
}

func assertSameArtifact(t *testing.T, label string, before, after json.RawMessage, o Options) {
	t.Helper()
	same, err := Equal(before, after, o)
	if err != nil {
		t.Fatalf("%s: comparing: %v", label, err)
	}
	if same {
		return
	}
	cb, _ := Canonicalize(before, o)
	ca, _ := Canonicalize(after, o)
	t.Errorf("%s: round trip changed the artifact\n  before: %s\n  after:  %s", label, cb, ca)
}

func readFixture(t *testing.T, path string) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return raw
}

// readDemoRule wraps a demo rule tree back into its payload, which is the only
// form of it that exists on disk.
func readDemoRule(t *testing.T, slug string) string {
	t.Helper()
	dir := filepath.Join("..", "..", "demo", "rules", "default_channel", slug)
	out, err := unwrap.WrapRule(dir, nil)
	if err != nil {
		t.Fatalf("wrapping demo rule %s: %v", slug, err)
	}
	return string(out)
}
