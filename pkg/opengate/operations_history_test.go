package opengate

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// operationJSON is the response shape taken from the platform's own example for
// /operation/jobs/{jobId}/operations, which the history search shares. The step
// carries "result" — not "status", which belongs to the operation.
const operationJSON = `{
  "operationId": "86dd3409-6fcd-49d4-be6b-b2fa497207ec",
  "jobId": "947eb1f6-f54b-11ed-b1ca-0242ac120008",
  "entityId": "device_1",
  "resourceType": "entity.device",
  "name": "DIAGNOSIS",
  "parameters": {"TYPE": "HARDWARE"},
  "attempts": {"total": 2, "current": 1},
  "notify": true,
  "execution": {
    "activatedDate": "2014-10-01T09:03:42Z",
    "startedDate": "2014-10-01T09:03:45Z",
    "finishedDate": "2014-10-01T09:04:45Z"
  },
  "user": "user@mail.com",
  "status": "FINISHED",
  "result": "SUCCESSFUL",
  "description": "successful operation",
  "date": "2014-10-01T09:04:45Z",
  "steps": [
    {"name": "PROVISION", "result": "SUCCESSFUL", "description": "ok; provisioned", "timestamp": "2012-09-27T16:46:02.10Z"},
    {"name": "PRESENCE", "result": "ERROR", "description": "no answer; timed out", "timestamp": "2012-09-27T16:46:03.10Z"},
    {"name": "MODEM", "result": "SKIPPED", "description": "skipped"},
    {"name": "REGISTER", "result": "NOT_EXECUTED", "description": "not executed"}
  ]
}`

func TestOperationDecoding(t *testing.T) {
	var op Operation
	if err := json.Unmarshal([]byte(operationJSON), &op); err != nil {
		t.Fatal(err)
	}

	if op.EntityID != "device_1" {
		t.Errorf("EntityID = %q", op.EntityID)
	}
	if op.Name != "DIAGNOSIS" {
		t.Errorf("Name = %q", op.Name)
	}
	if op.Status != "FINISHED" || op.Result != "SUCCESSFUL" {
		t.Errorf("Status/Result = %q/%q; the two are distinct fields", op.Status, op.Result)
	}
	if op.Execution == nil || op.Execution.StartedDate != "2014-10-01T09:03:45Z" {
		t.Errorf("execution.startedDate not decoded: %+v", op.Execution)
	}
	if op.Attempts == nil || op.Attempts.Total != 2 || op.Attempts.Current != 1 {
		t.Errorf("attempts not decoded: %+v", op.Attempts)
	}
	if len(op.Steps) != 4 {
		t.Fatalf("decoded %d steps, want 4", len(op.Steps))
	}

	// Every documented step result must round-trip, NOT_EXECUTED included — it is
	// the fourth value in the platform's enum.
	want := map[string]string{
		"PROVISION": StepResultSuccessful,
		"PRESENCE":  StepResultError,
		"MODEM":     StepResultSkipped,
		"REGISTER":  StepResultNotExecuted,
	}
	for name, result := range want {
		got, ok := op.StepResult(name)
		if !ok {
			t.Errorf("step %q missing", name)
			continue
		}
		if got != result {
			t.Errorf("step %q result = %q, want %q", name, got, result)
		}
	}
	if _, ok := op.StepResult("NOSUCHSTEP"); ok {
		t.Error("StepResult reported an absent step as present")
	}
}

// TestStepDescriptionSurvivesSemicolons is the reason this client asks for JSON
// and not the endpoint's text/plain form: the CSV separator collides with step
// descriptions, so the reference implementation had to repair rows with a regex.
func TestStepDescriptionSurvivesSemicolons(t *testing.T) {
	var op Operation
	if err := json.Unmarshal([]byte(operationJSON), &op); err != nil {
		t.Fatal(err)
	}
	got, _ := op.StepResult("PRESENCE")
	if got != StepResultError {
		t.Fatalf("PRESENCE result = %q", got)
	}
	for _, s := range op.Steps {
		if s.Name == "PRESENCE" && s.Description != "no answer; timed out" {
			t.Errorf("description = %q, want the semicolon preserved verbatim", s.Description)
		}
	}
}

func TestSearchOperationsHistoryRequestsJSON(t *testing.T) {
	resetTLS(t)

	var gotAccept, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"operations":[` + operationJSON + `],"page":{"number":1,"of":1}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	resp, err := c.SearchOperationsHistory(context.Background(), JobIDFilter("job-1"))
	if err != nil {
		t.Fatal(err)
	}

	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json (the CSV form is unusable)", gotAccept)
	}
	if gotPath != "/north/v80/search/entities/operations/history" {
		t.Errorf("path = %q", gotPath)
	}
	if len(resp.Operations) != 1 || resp.Operations[0].EntityID != "device_1" {
		t.Errorf("operations not decoded: %+v", resp.Operations)
	}
	if resp.Page == nil || resp.Page.Of != 1 {
		t.Errorf("page not decoded: %+v", resp.Page)
	}

	// The production filter shape, pinned.
	var filter map[string]any
	if err := json.Unmarshal([]byte(gotBody), &filter); err != nil {
		t.Fatalf("request body is not JSON: %v", err)
	}
	want := `{"filter":{"and":[{"eq":{"jobId":"job-1"}}]}}`
	var wantMap map[string]any
	_ = json.Unmarshal([]byte(want), &wantMap)
	gotJSON, _ := json.Marshal(filter)
	wantJSON, _ := json.Marshal(wantMap)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("filter = %s, want %s", gotJSON, wantJSON)
	}
}

func TestSearchOperationsHistoryEmptyResponse(t *testing.T) {
	resetTLS(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	resp, err := c.SearchOperationsHistory(context.Background(), nil)
	if err != nil {
		t.Fatalf("HTTP 204 must not be an error: %v", err)
	}
	if len(resp.Operations) != 0 {
		t.Errorf("expected no operations, got %d", len(resp.Operations))
	}
}

func TestSearchOperationsHistoryAllWalksPages(t *testing.T) {
	resetTLS(t)

	var starts []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Limit struct {
				Start int `json:"start"`
				Size  int `json:"size"`
			} `json:"limit"`
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &req)
		starts = append(starts, req.Limit.Start)

		ops := ""
		if req.Limit.Start <= 3 {
			ops = operationJSON + "," + operationJSON // exactly the page size
		}
		_, _ = w.Write([]byte(`{"operations":[` + ops + `],"page":{"number":` +
			strconv.Itoa(req.Limit.Start) + `,"of":3}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	filter, err := withPage(JobIDFilter("job-1"), 1, 2)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for op, err := range c.SearchOperationsHistoryAll(context.Background(), filter) {
		if err != nil {
			t.Fatal(err)
		}
		if op.JobID == "" {
			t.Error("operation decoded without a jobId")
		}
		count++
	}

	if count != 6 {
		t.Errorf("walked %d operations, want 6", count)
	}
	if len(starts) != 3 {
		t.Fatalf("issued %d requests, want 3: %v", len(starts), starts)
	}
	for i, s := range starts {
		if s != i+1 {
			t.Errorf("request %d asked for page %d, want %d", i, s, i+1)
		}
	}
}

func TestGetJobOperationsPageUsesQueryParams(t *testing.T) {
	resetTLS(t)

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"operations":[` + operationJSON + `],"page":{"number":1,"of":1}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")

	// This endpoint pages with query parameters, not a body limit, and caps at
	// 1000 rather than the 2000 of the filter-based searches.
	if _, err := c.GetJobOperationsPage(context.Background(), "job-1", 2, MaxJobOperationsPageSize+500); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "size=1000&start=2" {
		t.Errorf("query = %q, want size clamped to 1000 and start=2", gotQuery)
	}

	// A negative page omits start entirely, letting the platform serve its own
	// first page whichever number that is.
	if _, err := c.GetJobOperationsPage(context.Background(), "job-1", -1, 10); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "size=10" {
		t.Errorf("query = %q, want start omitted", gotQuery)
	}
}

// TestGetJobOperationsAllFollowsServerNumbering checks the walk adapts to a
// 0-based endpoint: start is documented with a default of 0 here, while the
// search endpoints count from 1, so the iterator follows the page numbers the
// server reports instead of assuming either convention.
func TestGetJobOperationsAllFollowsServerNumbering(t *testing.T) {
	resetTLS(t)

	var starts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		starts = append(starts, r.URL.Query().Get("start"))

		// A 0-based server: pages 0,1,2 are full, page 3 is empty.
		page := 0
		if s := r.URL.Query().Get("start"); s != "" {
			page, _ = strconv.Atoi(s)
		}
		ops := ""
		if page <= 2 {
			ops = operationJSON
		}
		_, _ = w.Write([]byte(`{"operations":[` + ops + `],"page":{"number":` + strconv.Itoa(page) + `,"of":3}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	count := 0
	for range c.GetJobOperationsAll(context.Background(), "job-1") {
		count++
	}

	// Page size is 1000 and each page returns 1 item, so the short-page rule
	// ends the walk after the first request.
	if count != 1 {
		t.Errorf("walked %d operations, want 1", count)
	}
	if len(starts) != 1 || starts[0] != "" {
		t.Errorf("first request start = %v, want it omitted", starts)
	}
}
