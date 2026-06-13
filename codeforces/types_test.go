package codeforces

import (
	"fmt"
	"testing"
)

func TestProblemURL(t *testing.T) {
	contestID := 1234
	index := "A"
	want := "https://codeforces.com/problemset/problem/1234/A"
	got := fmt.Sprintf("https://codeforces.com/problemset/problem/%d/%s", contestID, index)
	if got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		secs int
		want string
	}{
		{7200, "2h00m"},
		{9000, "2h30m"},
		{3600, "1h00m"},
		{5400, "1h30m"},
	}
	for _, tc := range cases {
		got := formatDuration(tc.secs)
		if got != tc.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}

func TestFormatStart(t *testing.T) {
	// 1704067200 = 2024-01-01 00:00:00 UTC
	got := formatStart(1704067200)
	want := "2024-01-01 00:00"
	if got != want {
		t.Errorf("formatStart(1704067200) = %q, want %q", got, want)
	}
}
