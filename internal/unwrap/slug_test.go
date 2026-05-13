package unwrap

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Operations", "operations"},
		{"My Cool Dashboard", "my-cool-dashboard"},
		{"  spaces   around  ", "spaces-around"},
		{"weird/chars:in@name", "weird-chars-in-name"},
		{"Acentúe esto: ñ", "acent-e-esto"},
		{"", "unnamed"},
		{"---", "unnamed"},
		{"42 things", "42-things"},
	}
	for _, c := range cases {
		got := Slugify(c.in)
		if got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDedupedSlug(t *testing.T) {
	taken := make(map[string]bool)

	a := DedupedSlug("Operations", "id-001", taken)
	if a != "operations" {
		t.Errorf("first call should return base slug, got %q", a)
	}

	b := DedupedSlug("Operations", "id-002", taken)
	if b == "operations" {
		t.Error("second call must not collide with first")
	}
	if !taken[b] {
		t.Errorf("dedup slug %q not marked as taken", b)
	}

	c := DedupedSlug("Unique Name", "id-003", taken)
	if c != "unique-name" {
		t.Errorf("non-colliding name should return base slug, got %q", c)
	}
}

func TestDedupedSlug_EmptyName(t *testing.T) {
	taken := make(map[string]bool)
	a := DedupedSlug("", "abc-123", taken)
	if a != "unnamed" {
		t.Errorf("empty name should yield 'unnamed', got %q", a)
	}
	b := DedupedSlug("", "abc-456", taken)
	if b == "unnamed" {
		t.Error("second empty-name workspace must not collide")
	}
}
