package opengate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestJobRequestMarshalsVerifiedShape pins the job body field by field against
// the shape observed in real payloads.
func TestJobRequestMarshalsVerifiedShape(t *testing.T) {
	active, notify, retries := false, true, 0
	req := JobRequest{
		Name:      "DIAGNOSIS",
		Active:    &active,
		Notify:    &notify,
		Callback:  "https://host/ruta",
		UserNotes: "",
		Schedule: &JobSchedule{
			Stop: &JobScheduleTime{Delayed: 600000},
			Scattering: &JobScattering{
				MaxSpread: 90,
				Strategy: &JobScatteringStrategy{
					Factor:         75,
					Field:          ScatteringFieldCellInfo,
					WarningMaxRate: 3,
				},
			},
		},
		OperationParameters: &OperationParams{
			Timeout:    60000,
			AckTimeout: 5000,
			Retries:    &retries,
		},
		Target: &JobTarget{Append: &JobTargetSet{Entities: []string{"dev-1"}}},
	}

	data, err := json.Marshal(JobContainer{Job: JobBody{Request: req}})
	if err != nil {
		t.Fatal(err)
	}

	want := `{"job":{"request":{
	  "name":"DIAGNOSIS",
	  "active": false,
	  "callback":"https://host/ruta",
	  "notify": true,
	  "schedule":{
	    "stop":{"delayed":600000},
	    "scattering":{
	      "maxSpread": 90,
	      "strategy":{"factor":75,"field":"subscription.collected.cellInfo","warningMaxRate":3}
	    }
	  },
	  "operationParameters":{"timeout":60000,"ackTimeout":5000,"retries":0},
	  "target":{"append":{"entities":["dev-1"]}}
	}}}`

	var got, expected any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatal(err)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(expected)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("job body mismatch\n got: %s\nwant: %s", gotJSON, wantJSON)
	}
}

// TestScatteringStrategyIsLowercase guards the documented trap: the published
// schema calls the field "Strategy", but the platform reads "strategy". A
// capitalised key is silently ignored, so the job runs unscattered.
func TestScatteringStrategyIsLowercase(t *testing.T) {
	data, err := json.Marshal(DefaultScattering())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"strategy"`) {
		t.Errorf("scattering must emit a lowercase \"strategy\" key, got %s", data)
	}
	if strings.Contains(string(data), `"Strategy"`) {
		t.Errorf("scattering must not emit a capitalised \"Strategy\" key, got %s", data)
	}
}

// TestActiveFalseSurvivesMarshalling is the subtle one: with a plain bool and
// omitempty, active:false would disappear and the job would be created ACTIVE,
// running the operation on a partial target.
func TestActiveFalseSurvivesMarshalling(t *testing.T) {
	active := false
	data, err := json.Marshal(JobRequest{Name: "X", Active: &active})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"active":false`) {
		t.Errorf("active:false must be emitted, got %s", data)
	}

	// And retries:0, for the same reason.
	retries := 0
	data, err = json.Marshal(OperationParams{Retries: &retries})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"retries":0`) {
		t.Errorf("retries:0 must be emitted, got %s", data)
	}
}

// TestJobRequestOmitsNameWhenEmpty matters because a PUT may only carry active,
// notify, callback, userNotes, schedule.start/stop and target.
func TestJobRequestOmitsNameWhenEmpty(t *testing.T) {
	data, err := json.Marshal(JobRequest{Target: &JobTarget{Append: &JobTargetSet{Entities: []string{"a"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"name"`) {
		t.Errorf("an update body must not carry a name, got %s", data)
	}
}

func TestMillisAcceptsNumberAndString(t *testing.T) {
	for _, raw := range []string{`600000`, `"600000"`} {
		var s JobScheduleTime
		if err := json.Unmarshal([]byte(`{"delayed":`+raw+`}`), &s); err != nil {
			t.Fatalf("delayed %s: %v", raw, err)
		}
		if s.Delayed != 600000 {
			t.Errorf("delayed %s decoded as %d", raw, s.Delayed)
		}
	}
	// Emitted as a number, which is what the schema declares.
	data, _ := json.Marshal(JobScheduleTime{Delayed: 600000})
	if string(data) != `{"delayed":600000}` {
		t.Errorf("delayed must marshal as a number, got %s", data)
	}
}

func TestJobRequestExtraEscapeHatch(t *testing.T) {
	req := JobRequest{
		Name:  "DIAGNOSIS",
		Extra: map[string]json.RawMessage{"futureField": json.RawMessage(`"v"`), "name": json.RawMessage(`"ignored"`)},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["futureField"] != "v" {
		t.Errorf("extra field missing: %s", data)
	}
	if m["name"] != "DIAGNOSIS" {
		t.Errorf("a typed field must win over Extra, got %v", m["name"])
	}
}

func TestScatteringValidate(t *testing.T) {
	tests := []struct {
		name    string
		s       *JobScattering
		wantErr bool
	}{
		{"nil is fine", nil, false},
		{"default", DefaultScattering(), false},
		{"at the ceiling", &JobScattering{MaxSpread: MaxScatteringSpread}, false},
		{"above the ceiling", &JobScattering{MaxSpread: 91}, true},
		{"100 is refused", &JobScattering{MaxSpread: 100}, true},
		{"negative", &JobScattering{MaxSpread: -1}, true},
		{"bad factor", &JobScattering{MaxSpread: 80, Strategy: &JobScatteringStrategy{Factor: 101}}, true},
		{"unsupported field", &JobScattering{MaxSpread: 80, Strategy: &JobScatteringStrategy{Field: "provision.device.identifier"}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.s.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected an error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestJobRequestValidate(t *testing.T) {
	many := make([]string, MaxJobTargetEntities+1)
	for i := range many {
		many[i] = fmt.Sprintf("dev-%d", i)
	}

	tests := []struct {
		name    string
		req     JobRequest
		wantErr bool
	}{
		{"needs a name", JobRequest{}, true},
		{"minimal", JobRequest{Name: "X"}, false},
		{"too many entities", JobRequest{Name: "X", Target: &JobTarget{Append: &JobTargetSet{Entities: many}}}, true},
		{"entities and tags mixed", JobRequest{Name: "X", Target: &JobTarget{
			Append: &JobTargetSet{Entities: []string{"a"}, Tags: []string{"t"}}}}, true},
		{"bad scattering", JobRequest{Name: "X", Schedule: &JobSchedule{Scattering: &JobScattering{MaxSpread: 100}}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected an error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// capturedRequest is one call a launch test server saw.
type capturedRequest struct {
	Method string
	Body   map[string]any
}

// launchServer records every request and answers with a job id.
func launchServer(t *testing.T, jobID string, calls *[]capturedRequest) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Errorf("unparseable body %q", raw)
			}
		}
		*calls = append(*calls, capturedRequest{Method: r.Method, Body: body})
		w.Header().Set("Location", "/north/v80/operation/jobs/"+jobID)
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"id":%q}`, jobID)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// request digs the job request object out of a captured body.
func (c capturedRequest) request(t *testing.T) map[string]any {
	t.Helper()
	job, ok := c.Body["job"].(map[string]any)
	if !ok {
		t.Fatalf("body has no job object: %v", c.Body)
	}
	req, ok := job["request"].(map[string]any)
	if !ok {
		t.Fatalf("job has no request object: %v", job)
	}
	return req
}

// appendedEntities returns target.append.entities of a captured body.
func (c capturedRequest) appendedEntities(t *testing.T) []string {
	t.Helper()
	req := c.request(t)
	target, ok := req["target"].(map[string]any)
	if !ok {
		return nil
	}
	app, ok := target["append"].(map[string]any)
	if !ok {
		return nil
	}
	raw, _ := app["entities"].([]any)
	out := make([]string, len(raw))
	for i, v := range raw {
		out[i], _ = v.(string)
	}
	return out
}

// TestLaunchJobBatches is the T4 acceptance criterion: 250 entities produce one
// creation with active:false and three appends of 100/100/50, with only the last
// carrying active:true.
func TestLaunchJobBatches(t *testing.T) {
	resetTLS(t)

	entities := make([]string, 250)
	for i := range entities {
		entities[i] = fmt.Sprintf("dev-%d", i)
	}

	var calls []capturedRequest
	srv := launchServer(t, "job-42", &calls)

	c := New(srv.URL, "tok")
	var progress []LaunchProgress
	res, err := c.LaunchJob(context.Background(),
		JobRequest{Name: "DIAGNOSIS", Schedule: &JobSchedule{Scattering: DefaultScattering()}},
		entities,
		LaunchOptions{OnProgress: func(p LaunchProgress) { progress = append(progress, p) }})
	if err != nil {
		t.Fatal(err)
	}

	if res.JobID != "job-42" {
		t.Errorf("job id = %q", res.JobID)
	}
	if res.Appended != 250 {
		t.Errorf("appended %d entities, want 250", res.Appended)
	}

	if len(calls) != 4 {
		t.Fatalf("issued %d requests, want 4 (1 create + 3 appends): %+v", len(calls), calls)
	}

	// 1. Creation: POST, active:false, no target.
	if calls[0].Method != http.MethodPost {
		t.Errorf("first call is %s, want POST", calls[0].Method)
	}
	create := calls[0].request(t)
	if create["active"] != false {
		t.Errorf("creation must carry active:false, got %v", create["active"])
	}
	if _, hasTarget := create["target"]; hasTarget {
		t.Errorf("creation must not carry a target, got %v", create["target"])
	}

	// 2. Three appends of 100/100/50, all PUT.
	wantSizes := []int{100, 100, 50}
	for i, want := range wantSizes {
		call := calls[i+1]
		if call.Method != http.MethodPut {
			t.Errorf("append %d is %s, want PUT", i+1, call.Method)
		}
		if got := len(call.appendedEntities(t)); got != want {
			t.Errorf("append %d carries %d entities, want %d", i+1, got, want)
		}
	}

	// 3. Only the last append activates the job.
	for i := range wantSizes {
		req := calls[i+1].request(t)
		active, present := req["active"]
		isLast := i == len(wantSizes)-1
		switch {
		case isLast && (!present || active != true):
			t.Errorf("the last append must carry active:true, got %v (present=%v)", active, present)
		case !isLast && present:
			t.Errorf("append %d must not touch active, got %v", i+1, active)
		}
	}

	// 4. Every entity appended exactly once, in order.
	var seen []string
	for i := range wantSizes {
		seen = append(seen, calls[i+1].appendedEntities(t)...)
	}
	if len(seen) != len(entities) {
		t.Fatalf("appended %d entities in total, want %d", len(seen), len(entities))
	}
	for i := range entities {
		if seen[i] != entities[i] {
			t.Errorf("entity %d = %q, want %q", i, seen[i], entities[i])
		}
	}

	// 5. The id is reported before the first batch, so a restart can resume.
	if len(progress) == 0 {
		t.Fatal("no progress reported")
	}
	if progress[0].JobID != "job-42" || progress[0].Appended != 0 || progress[0].Batch != 0 {
		t.Errorf("first progress must announce the id before any batch, got %+v", progress[0])
	}
	if last := progress[len(progress)-1]; !last.Active || last.Appended != 250 {
		t.Errorf("last progress = %+v, want active with everything appended", last)
	}
}

// TestLaunchJobExactBatchBoundary checks a multiple of the batch size does not
// produce a trailing empty append.
func TestLaunchJobExactBatchBoundary(t *testing.T) {
	resetTLS(t)

	entities := make([]string, 200)
	for i := range entities {
		entities[i] = fmt.Sprintf("dev-%d", i)
	}

	var calls []capturedRequest
	srv := launchServer(t, "job-1", &calls)

	c := New(srv.URL, "tok")
	if _, err := c.LaunchJob(context.Background(), JobRequest{Name: "X"}, entities, LaunchOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 {
		t.Fatalf("issued %d requests, want 3 (1 create + 2 appends): %d", len(calls), len(calls))
	}
	if req := calls[2].request(t); req["active"] != true {
		t.Errorf("the final append must activate the job, got %v", req["active"])
	}
}

func TestLaunchJobSingleBatchActivatesImmediately(t *testing.T) {
	resetTLS(t)

	var calls []capturedRequest
	srv := launchServer(t, "job-1", &calls)

	c := New(srv.URL, "tok")
	res, err := c.LaunchJob(context.Background(), JobRequest{Name: "X"}, []string{"a", "b"}, LaunchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("issued %d requests, want 2", len(calls))
	}
	if req := calls[1].request(t); req["active"] != true {
		t.Errorf("the only append must activate the job, got %v", req["active"])
	}
	if res.Appended != 2 {
		t.Errorf("appended %d, want 2", res.Appended)
	}
}

// TestLaunchJobCollectsRejections checks the entities the platform refuses are
// surfaced: they arrive inside successful responses.
func TestLaunchJobCollectsRejections(t *testing.T) {
	resetTLS(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"job-1","report":{"target":{
			"notProvisioned":["ghost-1"],"notAllowed":["forbidden-1"],"duplicated":["dup-1"]}}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	res, err := c.LaunchJob(context.Background(), JobRequest{Name: "X"}, []string{"a"}, LaunchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Rejected.Empty() {
		t.Fatal("rejections must be reported")
	}
	// The server repeats the same rejections on the create and the append; an
	// entity refused twice is still one refused entity, so they collapse.
	if res.Rejected.Total() != 3 {
		t.Errorf("collected %d rejections, want 3 deduplicated: %+v", res.Rejected.Total(), res.Rejected)
	}
	if len(res.Rejected.NotProvisioned) != 1 {
		t.Errorf("notProvisioned = %v, want a single entry", res.Rejected.NotProvisioned)
	}
	if res.Rejected.NotProvisioned[0] != "ghost-1" {
		t.Errorf("notProvisioned = %v", res.Rejected.NotProvisioned)
	}
}

func TestLaunchJobRejectsBadInput(t *testing.T) {
	resetTLS(t)

	c := New("https://example.invalid", "tok")

	if _, err := c.LaunchJob(context.Background(), JobRequest{Name: "X"}, nil, LaunchOptions{}); err == nil {
		t.Error("expected an error with no entities")
	}
	// A scattering above the ceiling must fail before any request is made.
	_, err := c.LaunchJob(context.Background(),
		JobRequest{Name: "X", Schedule: &JobSchedule{Scattering: &JobScattering{MaxSpread: 100}}},
		[]string{"a"}, LaunchOptions{})
	if err == nil {
		t.Error("expected an error for maxSpread above the ceiling")
	}
}

// TestLaunchJobStopsOnAppendError checks a failed batch reports how far it got,
// so the caller can resume instead of guessing.
func TestLaunchJobStopsOnAppendError(t *testing.T) {
	resetTLS(t)

	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 3 { // create, first append, then fail
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"boom"}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"job-1"}`))
	}))
	defer srv.Close()

	entities := make([]string, 150)
	for i := range entities {
		entities[i] = fmt.Sprintf("dev-%d", i)
	}

	c := New(srv.URL, "tok")
	res, err := c.LaunchJob(context.Background(), JobRequest{Name: "X"}, entities, LaunchOptions{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if res.JobID != "job-1" {
		t.Errorf("the job id must be reported even on failure, got %q", res.JobID)
	}
	if !strings.Contains(err.Error(), "job-1") || !strings.Contains(err.Error(), "100") {
		t.Errorf("the error must name the job and how far it got, got: %v", err)
	}
}

func TestPtr(t *testing.T) {
	// The point of Ptr is that a meaningful zero value survives omitempty.
	data, err := json.Marshal(JobRequest{Name: "X", Active: Ptr(false), Notify: Ptr(false)})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"active":false`, `"notify":false`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing %s in %s", want, data)
		}
	}
}
