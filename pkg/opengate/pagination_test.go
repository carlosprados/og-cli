package opengate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// recordedLimit is the limit block a test server saw on a request.
type recordedLimit struct {
	Start int
	Size  int
}

// pagedDeviceServer serves `pages` pages of `perPage` devices each. When
// reportOf is true the response carries page.of, mirroring the endpoints that
// report a total page count; when false it mirrors those that do not (e.g. the
// alarms search documents only page.number).
func pagedDeviceServer(t *testing.T, pages, perPage int, reportOf bool, seen *[]recordedLimit) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Limit recordedLimit `json:"limit"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("unparseable request body %q: %v", body, err)
		}
		*seen = append(*seen, req.Limit)

		page := req.Limit.Start
		if page < 1 {
			page = 1
		}

		devices := []json.RawMessage{}
		if page <= pages {
			for i := 0; i < perPage; i++ {
				id := fmt.Sprintf("dev-%d-%d", page, i)
				devices = append(devices, json.RawMessage(
					fmt.Sprintf(`{"provision.device.identifier":{"_value":{"_current":{"value":%q}}}}`, id)))
			}
		}

		resp := map[string]any{"devices": devices}
		info := map[string]any{"number": page}
		if reportOf {
			info["of"] = pages
		}
		resp["page"] = info

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestSearchDevicesAllWalksEveryPage is the T2 acceptance criterion: three pages
// served, every element traversed, and exactly three requests issued.
func TestSearchDevicesAllWalksEveryPage(t *testing.T) {
	resetTLS(t)

	var seen []recordedLimit
	srv := pagedDeviceServer(t, 3, 2, true, &seen)

	c := New(srv.URL, "tok")
	filter := json.RawMessage(`{"limit":{"size":2}}`)

	var got []string
	for dev, err := range c.SearchDevicesAll(context.Background(), filter) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, ExtractFlatValue(dev, "provision.device.identifier"))
	}

	want := []string{"dev-1-0", "dev-1-1", "dev-2-0", "dev-2-1", "dev-3-0", "dev-3-1"}
	if len(got) != len(want) {
		t.Fatalf("traversed %d devices, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("device %d = %q, want %q", i, got[i], want[i])
		}
	}

	if len(seen) != 3 {
		t.Fatalf("issued %d requests, want exactly 3: %+v", len(seen), seen)
	}
	for i, l := range seen {
		if l.Start != i+1 {
			t.Errorf("request %d asked for page %d, want %d (pagination is 1-based)", i, l.Start, i+1)
		}
		if l.Size != 2 {
			t.Errorf("request %d used size %d, want the caller's 2", i, l.Size)
		}
	}
}

// TestSearchDevicesAllStopsOnShortPage covers endpoints that never report
// page.of: a page shorter than the requested size is the end of the result set,
// and asking for one more page would be a wasted round trip.
func TestSearchDevicesAllStopsOnShortPage(t *testing.T) {
	resetTLS(t)

	var seen []recordedLimit
	// Page 3 exists but is short: 2 full pages of 2, then 1 item.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Limit recordedLimit `json:"limit"`
		}
		_ = json.Unmarshal(body, &req)
		seen = append(seen, req.Limit)

		counts := map[int]int{1: 2, 2: 2, 3: 1}
		devices := []json.RawMessage{}
		for i := 0; i < counts[req.Limit.Start]; i++ {
			devices = append(devices, json.RawMessage(`{}`))
		}
		// No "of" reported, on purpose.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"devices": devices,
			"page":    map[string]any{"number": req.Limit.Start},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	count := 0
	for _, err := range c.SearchDevicesAll(context.Background(), json.RawMessage(`{"limit":{"size":2}}`)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
	}

	if count != 5 {
		t.Errorf("traversed %d devices, want 5", count)
	}
	if len(seen) != 3 {
		t.Errorf("issued %d requests, want 3 (stop on the short page)", len(seen))
	}
}

// TestSearchDevicesAllStopsOnCancel is the other half of the T2 criterion: a ctx
// cancelled mid-iteration halts the walk and surfaces the reason, so a partial
// result is never mistaken for a complete one.
func TestSearchDevicesAllStopsOnCancel(t *testing.T) {
	resetTLS(t)

	var seen []recordedLimit
	srv := pagedDeviceServer(t, 10, 2, true, &seen)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // also covers the paths where the loop ends before cancelling
	c := New(srv.URL, "tok")

	var got int
	var iterErr error
	for _, err := range c.SearchDevicesAll(ctx, json.RawMessage(`{"limit":{"size":2}}`)) {
		if err != nil {
			iterErr = err
			break
		}
		got++
		if got == 3 { // part-way through page 2 of 10
			cancel()
		}
	}

	if !errors.Is(iterErr, context.Canceled) {
		t.Errorf("iteration error = %v, want it to wrap context.Canceled", iterErr)
	}
	if len(seen) >= 10 {
		t.Errorf("issued %d requests; cancellation should have stopped the walk early", len(seen))
	}
}

// TestSearchDevicesAllStopsWhenConsumerBreaks checks the iterator honours an
// early break instead of fetching the remaining pages.
func TestSearchDevicesAllStopsWhenConsumerBreaks(t *testing.T) {
	resetTLS(t)

	var seen []recordedLimit
	srv := pagedDeviceServer(t, 10, 2, true, &seen)

	c := New(srv.URL, "tok")
	for range c.SearchDevicesAll(context.Background(), json.RawMessage(`{"limit":{"size":2}}`)) {
		break
	}

	if len(seen) != 1 {
		t.Errorf("issued %d requests after an immediate break, want 1", len(seen))
	}
}

// TestSearchDevicesAllDefaultsPageSize checks the iterator picks DefaultPageSize
// when the caller's filter says nothing about paging.
func TestSearchDevicesAllDefaultsPageSize(t *testing.T) {
	resetTLS(t)

	var seen []recordedLimit
	srv := pagedDeviceServer(t, 1, 1, true, &seen)

	c := New(srv.URL, "tok")
	for range c.SearchDevicesAll(context.Background(), nil) {
	}

	if len(seen) == 0 {
		t.Fatal("no request issued")
	}
	if seen[0].Size != DefaultPageSize {
		t.Errorf("page size = %d, want DefaultPageSize (%d)", seen[0].Size, DefaultPageSize)
	}
}

func TestWithPagePreservesFilterAndSetsLimit(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		page   int
		size   int
		want   string
	}{
		{
			name:   "empty filter",
			filter: "",
			page:   1,
			size:   50,
			want:   `{"limit":{"start":1,"size":50}}`,
		},
		{
			name:   "preserves the filter clause",
			filter: `{"filter":{"eq":{"provision.device.identifier":"dev-1"}}}`,
			page:   3,
			size:   10,
			want:   `{"filter":{"eq":{"provision.device.identifier":"dev-1"}},"limit":{"start":3,"size":10}}`,
		},
		{
			name:   "overrides an existing limit",
			filter: `{"limit":{"size":999,"start":7}}`,
			page:   2,
			size:   5,
			want:   `{"limit":{"start":2,"size":5}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := withPage(json.RawMessage(tc.filter), tc.page, tc.size)
			if err != nil {
				t.Fatal(err)
			}
			// Compare semantically: map key order is not part of the contract.
			var gotMap, wantMap map[string]any
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("result is not valid JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(tc.want), &wantMap); err != nil {
				t.Fatal(err)
			}
			gotJSON, _ := json.Marshal(gotMap)
			wantJSON, _ := json.Marshal(wantMap)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("withPage = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestWithPageRejectsNonObjectFilter(t *testing.T) {
	if _, err := withPage(json.RawMessage(`["not","an","object"]`), 1, 10); err == nil {
		t.Error("expected an error for a filter that is not a JSON object")
	}
}

// TestSearchDevicesPageClampsSize checks the per-page method does not let a
// caller exceed the platform's documented maximum.
func TestSearchDevicesPageClampsSize(t *testing.T) {
	resetTLS(t)

	var seen []recordedLimit
	srv := pagedDeviceServer(t, 1, 1, true, &seen)

	c := New(srv.URL, "tok")
	if _, err := c.SearchDevicesPage(context.Background(), nil, 0, MaxPageSize+500); err != nil {
		t.Fatal(err)
	}

	if len(seen) != 1 {
		t.Fatalf("issued %d requests, want 1", len(seen))
	}
	if seen[0].Size != MaxPageSize {
		t.Errorf("size = %d, want it clamped to MaxPageSize (%d)", seen[0].Size, MaxPageSize)
	}
	if seen[0].Start != 1 {
		t.Errorf("start = %d, want page 0 normalised to 1", seen[0].Start)
	}
}
