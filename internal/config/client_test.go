package config

import "testing"

func TestClientOptionsFromProfile(t *testing.T) {
	// A profile with nothing set must add no options, so the library defaults
	// stay in force: no retrying and the default API version.
	if opts := (&Profile{}).ClientOptions(); len(opts) != 0 {
		t.Errorf("an empty profile produced %d options, want 0", len(opts))
	}
	if opts := (*Profile)(nil).ClientOptions(); opts != nil {
		t.Error("a nil profile must produce no options")
	}

	// Retries of 0 or 1 is not retrying, so it must not install a policy.
	for _, n := range []int{0, 1} {
		if opts := (&Profile{Retries: n}).ClientOptions(); len(opts) != 0 {
			t.Errorf("Retries=%d produced %d options, want 0", n, len(opts))
		}
	}

	got := (&Profile{APIVersion: "v81", Retries: 4}).ClientOptions()
	if len(got) != 2 {
		t.Errorf("api version + retries produced %d options, want 2", len(got))
	}
}
