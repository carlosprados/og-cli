package opengate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// Job limits and scattering defaults.
//
// MaxJobTargetEntities is the documented ceiling for one target list ("there is
// a limit in the number of entities in the array, by default the value is 100").
// A job body also has a size limit, 300 KBytes by default, which 100 identifiers
// never approach.
//
// The scattering values are ASSUMPTIONS, not facts: they are the platform's own
// example values plus the reasoning below, pending a real job body from the
// customer. Treat them as a starting point and revisit when one arrives.
const (
	MaxJobTargetEntities = 100

	// MaxScatteringSpread is a self-imposed ceiling, not a platform one.
	//
	// maxSpread is the percentage of the job's effective time over which the
	// operations are spread. At 100 the last operations are launched exactly as
	// the job's window expires, leaving them no time to run: they die without
	// being executed. The platform does not measure execution durations to
	// regress a safe last-launch instant, so a tail margin has to be reserved by
	// hand — and going above 90 spends margin we cannot prove we have.
	MaxScatteringSpread = 90

	// DefaultScatteringSpread leaves a 20% tail for in-flight operations to
	// drain. Two constraints pull in opposite directions:
	//
	//	dispatch rate < capacity:  window x maxSpread     > N x timeout / threads
	//	drain margin:              window x (1-maxSpread) >= 10% and >> timeout
	//
	// Raising maxSpread smooths the dispatch rate (kinder to a mobile cell) but
	// eats the drain margin. So when the real per-device timeout turns out to be
	// larger, grow the window — never shrink the tail.
	DefaultScatteringSpread = 80

	// DefaultScatteringFactor is the value used by every scattering example in
	// the platform documentation.
	DefaultScatteringFactor = 75

	// DefaultWarningMaxRate is the platform examples' value, in operations per
	// second. It is a speed check, so it only needs to sit above the real
	// dispatch rate.
	DefaultWarningMaxRate = 3

	// ScatteringFieldCellInfo is the only field scattering accepts today.
	ScatteringFieldCellInfo = "subscription.collected.cellInfo"
)

// Millis is a duration in milliseconds.
//
// It marshals as a JSON number, which is what the platform schema declares and
// what the jobs this client already launches successfully use. It unmarshals
// from either a number or a string, because quoted values have been reported in
// the wild and a type error on a response would be a pointless failure.
type Millis int64

func (m Millis) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(m), 10)), nil
}

func (m *Millis) UnmarshalJSON(data []byte) error {
	s := string(bytes.Trim(data, `"`))
	if s == "" || s == "null" {
		*m = 0
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing milliseconds from %s: %w", data, err)
	}
	*m = Millis(v)
	return nil
}

// Ptr returns a pointer to v.
//
// Several job fields are pointers because their zero value is meaningful and
// would otherwise be dropped by omitempty — active:false and retries:0 in
// particular. This spares every caller a temporary variable:
//
//	req.Active = opengate.Ptr(false)
func Ptr[T any](v T) *T { return &v }

// JobContainer is the top-level body of a job request: {"job":{"request":{...}}}.
type JobContainer struct {
	Job JobBody `json:"job"`
}

// JobBody wraps a job request.
type JobBody struct {
	Request JobRequest `json:"request"`
}

// JobRequest is a job definition.
//
// Active and Notify are pointers on purpose: false is a meaningful value and
// would vanish under omitempty. Getting that wrong on Active is not a cosmetic
// bug — a job meant to be created inactive would go live immediately, running
// the operation on whatever partial target it had at that moment.
type JobRequest struct {
	// Name is omitted when empty so the same struct can build a PUT, which only
	// accepts active, notify, callback, userNotes, schedule.start, schedule.stop
	// and target — sending name there risks a "Forbidden field" rejection.
	Name                string           `json:"name,omitempty"`
	Active              *bool            `json:"active,omitempty"`
	Notify              *bool            `json:"notify,omitempty"`
	Callback            string           `json:"callback,omitempty"`
	UserNotes           string           `json:"userNotes,omitempty"`
	Parameters          json.RawMessage  `json:"parameters,omitempty"`
	Schedule            *JobSchedule     `json:"schedule,omitempty"`
	OperationParameters *OperationParams `json:"operationParameters,omitempty"`
	Target              *JobTarget       `json:"target,omitempty"`

	// Extra carries fields the platform accepts but these structs do not model,
	// so a new API field never forces a library change. Its keys are merged into
	// the request object and lose to the typed fields on collision.
	Extra map[string]json.RawMessage `json:"-"`
}

// MarshalJSON merges Extra into the request object.
func (r JobRequest) MarshalJSON() ([]byte, error) {
	type plain JobRequest // avoid recursing into this method
	data, err := json.Marshal(plain(r))
	if err != nil {
		return nil, err
	}
	if len(r.Extra) == 0 {
		return data, nil
	}

	var merged map[string]json.RawMessage
	if err := json.Unmarshal(data, &merged); err != nil {
		return nil, err
	}
	for k, v := range r.Extra {
		if _, taken := merged[k]; !taken {
			merged[k] = v
		}
	}
	return json.Marshal(merged)
}

// JobSchedule is the scheduling block of a job request.
type JobSchedule struct {
	Start      *JobScheduleTime `json:"start,omitempty"`
	Stop       *JobScheduleTime `json:"stop,omitempty"`
	Scattering *JobScattering   `json:"scattering,omitempty"`
	Window     json.RawMessage  `json:"window,omitempty"`
}

// JobScheduleTime is a schedule bound: a delay or an absolute date.
type JobScheduleTime struct {
	Delayed Millis `json:"delayed,omitempty"`
	Date    string `json:"date,omitempty"`
}

// JobScattering disperses a job's operations over its effective time.
type JobScattering struct {
	MaxSpread int `json:"maxSpread"`
	// Strategy is lowercase on the wire. The published schema names it
	// "Strategy" with a capital S, but the examples and the running platform use
	// "strategy" — a mismatch that only shows up as a silently ignored strategy.
	Strategy *JobScatteringStrategy `json:"strategy,omitempty"`
}

// JobScatteringStrategy defines how the dispersion is grouped.
type JobScatteringStrategy struct {
	Factor         int    `json:"factor"`
	Field          string `json:"field,omitempty"`
	WarningMaxRate int    `json:"warningMaxRate,omitempty"`
}

// DefaultScattering returns the documented-assumption scattering block: spread
// over 80% of the window, leaving a 20% tail for operations to drain.
func DefaultScattering() *JobScattering {
	return &JobScattering{
		MaxSpread: DefaultScatteringSpread,
		Strategy: &JobScatteringStrategy{
			Factor:         DefaultScatteringFactor,
			Field:          ScatteringFieldCellInfo,
			WarningMaxRate: DefaultWarningMaxRate,
		},
	}
}

// Validate rejects a scattering configuration that cannot work.
func (s *JobScattering) Validate() error {
	if s == nil {
		return nil
	}
	if s.MaxSpread < 0 || s.MaxSpread > MaxScatteringSpread {
		return fmt.Errorf(
			"scattering maxSpread is %d: it must be between 0 and %d — at a higher spread the "+
				"last operations are launched as the job window expires and never run, and the "+
				"platform does not measure durations to reserve that margin for you",
			s.MaxSpread, MaxScatteringSpread)
	}
	if s.Strategy != nil {
		if s.Strategy.Factor < 0 || s.Strategy.Factor > 100 {
			return fmt.Errorf("scattering strategy factor is %d: it must be between 0 and 100", s.Strategy.Factor)
		}
		if s.Strategy.Field != "" && s.Strategy.Field != ScatteringFieldCellInfo {
			return fmt.Errorf("scattering strategy field %q is not supported; the only accepted value is %q",
				s.Strategy.Field, ScatteringFieldCellInfo)
		}
	}
	return nil
}

// OperationParams configures the operation sent to each target.
//
// Retries is a pointer because 0 — do not retry — is both meaningful and the
// value you usually want on unreachable devices, where a retry only doubles the
// wall-clock to reach the same answer.
type OperationParams struct {
	Timeout         Millis   `json:"timeout,omitempty"`
	AckTimeout      Millis   `json:"ackTimeout,omitempty"`
	Retries         *int     `json:"retries,omitempty"`
	RetriesDelay    Millis   `json:"retriesDelay,omitempty"`
	RetryResultList []string `json:"retryResultList,omitempty"`
}

// JobTarget selects the entities a job acts on.
type JobTarget struct {
	Append *JobTargetSet `json:"append,omitempty"`
	Remove *JobTargetSet `json:"remove,omitempty"`
}

// JobTargetSet is a set of entities or tags. Entities and tags cannot be mixed.
type JobTargetSet struct {
	Entities []string `json:"entities,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

// JobTargetRejections lists the entities the platform refused to associate with
// a job. They are reported inside a successful response, so a caller that only
// checks the status code launches a job on fewer entities than it asked for and
// never finds out.
type JobTargetRejections struct {
	NotProvisioned []string `json:"notProvisioned,omitempty"`
	NotAllowed     []string `json:"notAllowed,omitempty"`
	Duplicated     []string `json:"duplicated,omitempty"`
}

// Total counts every rejected entity.
func (r JobTargetRejections) Total() int {
	return len(r.NotProvisioned) + len(r.NotAllowed) + len(r.Duplicated)
}

// Empty reports whether the platform accepted every entity.
func (r JobTargetRejections) Empty() bool { return r.Total() == 0 }

// merge accumulates another response's rejections, without repeating an entity
// that more than one response reported.
func (r *JobTargetRejections) merge(o JobTargetRejections) {
	r.NotProvisioned = appendUnique(r.NotProvisioned, o.NotProvisioned)
	r.NotAllowed = appendUnique(r.NotAllowed, o.NotAllowed)
	r.Duplicated = appendUnique(r.Duplicated, o.Duplicated)
}

func appendUnique(dst, src []string) []string {
	if len(src) == 0 {
		return dst
	}
	seen := make(map[string]bool, len(dst))
	for _, v := range dst {
		seen[v] = true
	}
	for _, v := range src {
		if !seen[v] {
			seen[v] = true
			dst = append(dst, v)
		}
	}
	return dst
}

// jobResponse is the part of a job response this client interprets.
type jobResponse struct {
	ID     string `json:"id"`
	Report struct {
		Target JobTargetRejections `json:"target"`
	} `json:"report"`
}

// Validate checks a job request for the mistakes that only surface against a
// live platform.
func (r JobRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("job request name is required")
	}
	if r.Schedule != nil {
		if err := r.Schedule.Scattering.Validate(); err != nil {
			return err
		}
	}
	if r.Target != nil {
		for _, set := range []*JobTargetSet{r.Target.Append, r.Target.Remove} {
			if set == nil {
				continue
			}
			if len(set.Entities) > MaxJobTargetEntities {
				return fmt.Errorf("target list has %d entities, more than the %d a job accepts; "+
					"use LaunchJob to append them in batches", len(set.Entities), MaxJobTargetEntities)
			}
			if len(set.Entities) > 0 && len(set.Tags) > 0 {
				return fmt.Errorf("a target set cannot mix entities and tags")
			}
		}
	}
	return nil
}

// CreateJobRequest creates a job from a typed request, returning the assigned id
// and any entities the platform refused.
func (c *Client) CreateJobRequest(ctx context.Context, req JobRequest) (string, JobTargetRejections, error) {
	if err := req.Validate(); err != nil {
		return "", JobTargetRejections{}, err
	}
	body, err := json.Marshal(JobContainer{Job: JobBody{Request: req}})
	if err != nil {
		return "", JobTargetRejections{}, fmt.Errorf("marshaling job: %w", err)
	}

	data, err := c.CreateJob(ctx, body)
	if err != nil {
		return "", JobTargetRejections{}, err
	}

	var resp jobResponse
	if len(data) > 0 {
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", JobTargetRejections{}, fmt.Errorf("parsing job response: %w", err)
		}
	}
	return resp.ID, resp.Report.Target, nil
}

// LaunchProgress reports the state of a batched launch.
type LaunchProgress struct {
	JobID    string `json:"jobId"`
	Appended int    `json:"appended"` // entities appended so far
	Total    int    `json:"total"`    // entities requested
	Batch    int    `json:"batch"`    // 1-based batch number, 0 for the creation step
	Active   bool   `json:"active"`
}

// LaunchOptions configures LaunchJob.
type LaunchOptions struct {
	// BatchSize caps each append; it is clamped to MaxJobTargetEntities.
	BatchSize int
	// OnProgress, when set, is called after the job is created — before the
	// first batch, so the id can be persisted and a restart can resume — and
	// after every batch.
	OnProgress func(LaunchProgress)
}

// LaunchResult reports the outcome of a batched launch.
type LaunchResult struct {
	JobID    string              `json:"jobId"`
	Appended int                 `json:"appended"`
	Rejected JobTargetRejections `json:"rejected,omitzero"`
}

// LaunchJob runs the pattern the platform prescribes for a fleet larger than one
// target list: create the job inactive, append the entities in batches, and
// activate it.
//
// Activation is merged into the last batch's PUT, so the job is never active
// with a partial target — a job that goes live early runs the operation on
// whichever entities happened to be attached at that moment, and there is no
// undo for an operation already sent to a device.
//
// The id is reported through OnProgress as soon as it exists, before the first
// batch, so a caller can persist it and resume instead of orphaning a
// half-populated job on restart.
//
// Entities the platform refuses are collected in the result: they arrive inside
// successful responses, so a caller checking only for errors would silently
// operate on fewer devices than it asked for.
func (c *Client) LaunchJob(ctx context.Context, req JobRequest, entities []string, opts LaunchOptions) (LaunchResult, error) {
	var res LaunchResult

	if len(entities) == 0 {
		return res, fmt.Errorf("no entities to launch on")
	}
	batch := opts.BatchSize
	if batch <= 0 || batch > MaxJobTargetEntities {
		batch = MaxJobTargetEntities
	}

	// Create inactive and without a target: the platform allows an inactive job
	// with no entities, and an active one with none is a 400.
	create := req
	inactive := false
	create.Active = &inactive
	create.Target = nil

	if err := create.Validate(); err != nil {
		return res, err
	}

	jobID, rejected, err := c.CreateJobRequest(ctx, create)
	if err != nil {
		return res, err
	}
	if jobID == "" {
		return res, fmt.Errorf("job created but the platform returned no id")
	}
	res.JobID = jobID
	res.Rejected.merge(rejected)

	if opts.OnProgress != nil {
		opts.OnProgress(LaunchProgress{JobID: jobID, Total: len(entities)})
	}

	for i, n := 0, 0; i < len(entities); n++ {
		end := min(i+batch, len(entities))
		last := end == len(entities)

		// Only the fields a PUT is allowed to change: the target, and active on
		// the final batch.
		update := JobRequest{
			Target: &JobTarget{Append: &JobTargetSet{Entities: entities[i:end]}},
		}
		if last {
			active := true
			update.Active = &active
		}

		body, err := json.Marshal(JobContainer{Job: JobBody{Request: update}})
		if err != nil {
			return res, fmt.Errorf("marshaling batch %d: %w", n+1, err)
		}

		data, err := c.UpdateJob(ctx, jobID, body)
		if err != nil {
			return res, fmt.Errorf("appending batch %d of job %s (%d entities already appended): %w",
				n+1, jobID, res.Appended, err)
		}
		if len(data) > 0 {
			var resp jobResponse
			if err := json.Unmarshal(data, &resp); err == nil {
				res.Rejected.merge(resp.Report.Target)
			}
		}

		res.Appended += end - i
		if opts.OnProgress != nil {
			opts.OnProgress(LaunchProgress{
				JobID:    jobID,
				Appended: res.Appended,
				Total:    len(entities),
				Batch:    n + 1,
				Active:   last,
			})
		}
		i = end
	}

	return res, nil
}
