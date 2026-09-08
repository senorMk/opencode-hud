package store

import (
	"strings"
	"testing"
)

func TestDisplayTitle(t *testing.T) {
	// Auto title: raw ISO must be replaced by a short local date.
	got := DisplayTitle("New session - 2026-09-08T13:39:38.688Z")
	if !strings.HasPrefix(got, "New session · Sep 8, ") {
		t.Errorf("DisplayTitle(auto) = %q, want prefix %q", got, "New session · Sep 8, ")
	}
	if strings.Contains(got, "2026-09-08T13:39") {
		t.Errorf("DisplayTitle(auto) = %q, still contains raw ISO", got)
	}
	cases := []struct {
		in   string
		want string
	}{
		{"Fix the login bug", "Fix the login bug"},
		{"", "Untitled session"},
		{"New session - not-a-date", "New session - not-a-date"},
	}
	for _, c := range cases {
		if got := DisplayTitle(c.in); got != c.want {
			t.Errorf("DisplayTitle(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormaliseModel(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`{"id":"muse-spark-1.3","providerID":"opencode"}`, "opencode/muse-spark-1.3"},
		{`{"id":"m"}`, "m"},
		{"ollama/qwen3", "ollama/qwen3"},
		{"", ""},
	}
	for _, c := range cases {
		if got := NormaliseModel(c.in); got != c.want {
			t.Errorf("NormaliseModel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
