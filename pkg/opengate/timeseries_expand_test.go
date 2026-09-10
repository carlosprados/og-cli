package opengate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The list endpoint's expansions are opt-in and whitelisted per build. Asking
// for sorts on an instance that does not know the value gets the whole request
// rejected with HTTP 400, not a response without sorts — so between v2.6.0 and
// this fix `og timeseries list` returned nothing at all on such an instance
// (reported live against an on-premises v80 tenant, 2026-09-10).
const tsListBody = `{"timeseries":[{"identifier":"ts-1","name":"ConsumoDiarioMeter"}]}`

// tsListServer serves the list endpoint, rejecting any request whose expand
// carries a value outside accepts, and records every expand it was asked for.
func tsListServer(t *testing.T, accepts map[string]bool) (*httptest.Server, *[]string) {
	t.Helper()
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expand := r.URL.Query().Get("expand")
		seen = append(seen, expand)
		for _, v := range strings.Split(expand, ",") {
			if !accepts[v] {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(expandRejectedBody))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(tsListBody))
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
}

func TestListTimeSeriesAsksForSortsFirst(t *testing.T) {
	srv, seen := tsListServer(t, map[string]bool{"columns": true, "context": true, "sorts": true})

	resp, err := New(srv.URL, "token").ListTimeSeries(context.Background(), "MRG")
	if err != nil {
		t.Fatalf("ListTimeSeries: %v", err)
	}
	if len(resp.Timeseries) != 1 {
		t.Fatalf("got %d time series, want 1", len(resp.Timeseries))
	}
	// One round trip, and it asked for everything: an instance that supports
	// sorts must never pay for the fallback.
	if len(*seen) != 1 || (*seen)[0] != tsListExpand {
		t.Errorf("expand requests = %v, want [%s]", *seen, tsListExpand)
	}
}

func TestListTimeSeriesDegradesWhenSortsRejected(t *testing.T) {
	srv, seen := tsListServer(t, map[string]bool{"columns": true, "context": true})

	resp, err := New(srv.URL, "token").ListTimeSeries(context.Background(), "MRG")
	if err != nil {
		t.Fatalf("ListTimeSeries on an instance without sorts: %v", err)
	}
	if len(resp.Timeseries) != 1 || resp.Timeseries[0].Name != "ConsumoDiarioMeter" {
		t.Fatalf("got %+v, want the one time series", resp.Timeseries)
	}
	want := []string{tsListExpand, tsListExpandLegacy}
	if len(*seen) != 2 || (*seen)[0] != want[0] || (*seen)[1] != want[1] {
		t.Errorf("expand requests = %v, want %v", *seen, want)
	}
}

func TestListTimeSeriesRawDegradesToo(t *testing.T) {
	srv, seen := tsListServer(t, map[string]bool{"columns": true, "context": true})

	raw, err := New(srv.URL, "token").ListTimeSeriesRaw(context.Background(), "MRG")
	if err != nil {
		t.Fatalf("ListTimeSeriesRaw on an instance without sorts: %v", err)
	}
	if string(raw) != tsListBody {
		t.Errorf("raw = %s, want the platform's bytes", raw)
	}
	if len(*seen) != 2 {
		t.Errorf("expand requests = %v, want two", *seen)
	}
}

// A 400 that is not about expand must surface, not be retried away: the retry
// would hide it behind a second, identically failing request.
func TestListTimeSeriesDoesNotRetryUnrelated400(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"code":"0xA0","message":"Invalid query parameters.","context":[{"value":"MRG","name":"organization"}]}]}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL, "token").ListTimeSeries(context.Background(), "MRG")
	if err == nil {
		t.Fatal("ListTimeSeries: want an error, got none")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry on an unrelated 400)", calls)
	}
}
