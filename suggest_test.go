package main

import "testing"

func TestEditDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"dev", "dev", 0},
		{"dev", "de", 1},
		{"buld", "build", 1},
		{"dve", "dev", 1},
		{"dev", "deploy", 4},
	}
	for _, c := range cases {
		if got := editDistance(c.a, c.b); got != c.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestClosestMatch(t *testing.T) {
	candidates := []string{"build", "dev", "deploy"}

	if got := closestMatch("buld", candidates); got != "build" {
		t.Errorf("got %q, want build", got)
	}
	if got := closestMatch("de", candidates); got != "dev" {
		t.Errorf("got %q, want dev", got)
	}
	if got := closestMatch("dve", candidates); got != "dev" {
		t.Errorf("got %q, want dev", got)
	}
	// Nothing close enough
	if got := closestMatch("zzzzzz", candidates); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	// Exact case-insensitive match is not a typo
	if got := closestMatch("DEV", candidates); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
