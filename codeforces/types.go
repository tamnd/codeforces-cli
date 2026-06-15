package codeforces

import (
	"fmt"
	"time"
)

// Problem is the record emitted for Codeforces problemset entries.
type Problem struct {
	Rank   int    `json:"rank"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Rating int    `json:"rating"`
	Tags   string `json:"tags"`
	Solved int    `json:"solved"`
	URL    string `json:"url"`
}

// User is the record emitted for Codeforces user profiles.
type User struct {
	Handle  string `json:"handle"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Rank    string `json:"rank"`
	Rating  int    `json:"rating"`
	MaxRat  int    `json:"max_rating"`
	Friends int    `json:"friends"`
	URL     string `json:"url"`
}

// Contest is the record emitted for Codeforces contests.
type Contest struct {
	Rank     int    `json:"rank"`
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Phase    string `json:"phase"`
	Duration string `json:"duration"`
	Start    string `json:"start"`
	URL      string `json:"url"`
}

// Tag is the record emitted for Codeforces problem tags.
type Tag struct {
	Name string `json:"name"`
}

// formatDuration converts durationSeconds to a human-readable string like "2h00m".
func formatDuration(d int) string {
	return fmt.Sprintf("%dh%02dm", d/3600, (d%3600)/60)
}

// formatStart formats a Unix timestamp as "2006-01-02 15:04" in UTC.
// Returns "" if ts == 0.
func formatStart(unix int64) string {
	if unix == 0 {
		return ""
	}
	return time.Unix(unix, 0).UTC().Format("2006-01-02 15:04")
}

// knownTags is the comprehensive list of Codeforces problem tags.
var knownTags = []string{
	"2-sat",
	"binary search",
	"bitmasks",
	"brute force",
	"chinese remainder theorem",
	"combinatorics",
	"constructive algorithms",
	"data structures",
	"dfs and similar",
	"divide and conquer",
	"dp",
	"dsu",
	"expression parsing",
	"fft",
	"flows",
	"games",
	"geometry",
	"graph matchings",
	"graphs",
	"greedy",
	"hashing",
	"implementation",
	"interactive",
	"math",
	"matrices",
	"meet-in-the-middle",
	"number theory",
	"probabilities",
	"schedules",
	"shortest paths",
	"sortings",
	"special",
	"string suffix structures",
	"strings",
	"ternary search",
	"trees",
	"two pointers",
}

// Tags returns the static list of known Codeforces problem tags.
// No network call is made. The list is sorted alphabetically.
func Tags() []Tag {
	tags := make([]Tag, len(knownTags))
	for i, t := range knownTags {
		tags[i] = Tag{Name: t}
	}
	return tags
}
