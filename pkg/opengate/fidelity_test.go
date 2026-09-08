package opengate

import (
	"encoding/json"
	"testing"
)

// The fixtures here are unedited responses from a live OpenGate instance
// (datamodel_notfilterable.json is one, narrowed to the datastreams that carry
// the field, because the full document is 60 KB of noise for this assertion).
//
// They exist because three fields the platform returns on every request were
// absent from these structs and were therefore dropped on every decode:
// `indexed` and `notFilterable` on a datastream, and a dataset's entire `sorts`
// catalogue. None of the three appears in the published OpenAPI spec under
// ogdoc/, so modelling the spec was not enough — these tests pin the shape the
// server actually sends.

// TestDatastreamKeepsUndocumentedFlags asserts that a datamodel survives a
// decode/encode cycle with `indexed` intact, and that the flag round-trips by
// value rather than being flattened to a default.
func TestDatastreamKeepsUndocumentedFlags(t *testing.T) {
	var dm Datamodel
	if err := json.Unmarshal(loadFixture(t, "datamodel_get.json"), &dm); err != nil {
		t.Fatalf("unmarshal datamodel: %v", err)
	}

	var checked int
	for _, cat := range dm.Categories {
		for _, ds := range cat.Datastreams {
			if ds.Indexed == nil {
				t.Errorf("datastream %s: indexed was dropped on decode", ds.Identifier)
				continue
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("fixture carried no datastream with indexed; it cannot prove anything")
	}

	// Re-encoding must keep the field, which is where omitempty on a plain bool
	// would have silently lost every `indexed: false`.
	out, err := json.Marshal(&dm)
	if err != nil {
		t.Fatalf("marshal datamodel: %v", err)
	}
	if got := countKey(t, out, "indexed"); got != checked {
		t.Errorf("indexed survived decode but not encode: %d of %d datastreams kept it", got, checked)
	}
}

// TestDatastreamKeepsNotFilterable covers the flag that appears on only a
// handful of datastreams platform-wide, so a fixture without it would pass
// vacuously.
func TestDatastreamKeepsNotFilterable(t *testing.T) {
	var dm Datamodel
	if err := json.Unmarshal(loadFixture(t, "datamodel_notfilterable.json"), &dm); err != nil {
		t.Fatalf("unmarshal datamodel: %v", err)
	}

	var found int
	for _, cat := range dm.Categories {
		for _, ds := range cat.Datastreams {
			if ds.NotFilterable == nil {
				t.Errorf("datastream %s: notFilterable was dropped on decode", ds.Identifier)
				continue
			}
			if !*ds.NotFilterable {
				t.Errorf("datastream %s: notFilterable decoded as false, fixture says true", ds.Identifier)
			}
			found++
		}
	}
	if found == 0 {
		t.Fatal("fixture carried no datastream with notFilterable")
	}

	out, err := json.Marshal(&dm)
	if err != nil {
		t.Fatalf("marshal datamodel: %v", err)
	}
	if got := countKey(t, out, "notFilterable"); got != found {
		t.Errorf("notFilterable count changed across round-trip: got %d, want %d", got, found)
	}
}

// TestDatasetKeepsSorts is the regression for the worst of the three: a dataset
// read through this struct used to come back without any of its named sorts.
func TestDatasetKeepsSorts(t *testing.T) {
	fixture := loadFixture(t, "dataset_get.json")

	var ds Dataset
	if err := json.Unmarshal(fixture, &ds); err != nil {
		t.Fatalf("unmarshal dataset: %v", err)
	}

	if len(ds.Sorts) == 0 {
		t.Fatal("sorts was dropped on decode")
	}

	// A sort is only useful if its columns and direction came with it.
	var sawDerived, sawDeclared bool
	for _, s := range ds.Sorts {
		if s.Identifier == "" {
			t.Error("sort decoded without an identifier")
		}
		if len(s.Columns) == 0 {
			t.Errorf("sort %s decoded with no columns", s.Identifier)
		}
		for _, col := range s.Columns {
			if col.Name == "" || col.Direction == "" {
				t.Errorf("sort %s: column decoded as %+v, missing name or direction", s.Identifier, col)
			}
		}
		if s.Derived {
			sawDerived = true
		} else {
			sawDeclared = true
		}
	}
	if !sawDerived || !sawDeclared {
		t.Error("fixture should cover both derived and declared sorts")
	}

	// The whole point is fidelity, so compare against the fixture rather than
	// trusting the field count: every sort the server sent must come back.
	var raw struct {
		Sorts []json.RawMessage `json:"sorts"`
	}
	if err := json.Unmarshal(fixture, &raw); err != nil {
		t.Fatalf("unmarshal fixture sorts: %v", err)
	}
	if len(ds.Sorts) != len(raw.Sorts) {
		t.Errorf("kept %d sorts, server sent %d", len(ds.Sorts), len(raw.Sorts))
	}
}

// countKey counts how many times a key appears anywhere in a JSON document.
func countKey(t *testing.T, data []byte, key string) int {
	t.Helper()
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("counting %q: %v", key, err)
	}
	var walk func(any) int
	walk = func(n any) int {
		var total int
		switch t := n.(type) {
		case map[string]any:
			for k, child := range t {
				if k == key {
					total++
				}
				total += walk(child)
			}
		case []any:
			for _, child := range t {
				total += walk(child)
			}
		}
		return total
	}
	return walk(v)
}

// TestTimeSeriesKeepsSortMetadata pins the two fields a sort carries beyond its
// columns. They were missing from TSSort, so every `og ts get` came back with
// eight sorts stripped of their descriptions and of the flag saying which ones
// the platform generated itself.
func TestTimeSeriesKeepsSortMetadata(t *testing.T) {
	fixture := loadFixture(t, "timeseries_get.json")

	var ts TimeSeries
	if err := json.Unmarshal(fixture, &ts); err != nil {
		t.Fatalf("unmarshal timeseries: %v", err)
	}
	if len(ts.Sorts) == 0 {
		t.Fatal("sorts was dropped on decode")
	}

	var sawDerived, sawDeclared bool
	for _, srt := range ts.Sorts {
		if srt.Identifier == "" {
			t.Error("sort decoded without an identifier")
		}
		if srt.Description == "" {
			t.Errorf("sort %s: description was dropped", srt.Identifier)
		}
		if len(srt.Columns) == 0 {
			t.Errorf("sort %s: columns were dropped", srt.Identifier)
		}
		if srt.Derived {
			sawDerived = true
		} else {
			sawDeclared = true
		}
	}
	if !sawDerived || !sawDeclared {
		t.Error("fixture should cover both derived and declared sorts")
	}

	// derived:false must survive re-encoding — omitempty on a plain bool is
	// exactly how it went missing before.
	out, err := json.Marshal(&ts)
	if err != nil {
		t.Fatalf("marshal timeseries: %v", err)
	}
	if got, want := countKey(t, out, "derived"), len(ts.Sorts); got != want {
		t.Errorf("derived survived decode but not encode: %d of %d sorts kept it", got, want)
	}
	if got, want := countKey(t, out, "description"), countKey(t, fixture, "description"); got != want {
		t.Errorf("description count changed across round-trip: got %d, want %d", got, want)
	}
}
