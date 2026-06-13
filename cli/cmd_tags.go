package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/codeforces-cli/codeforces"
)

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
	"shortest paths",
	"sortings",
	"special",
	"string suffix structures",
	"strings",
	"ternary search",
	"trees",
	"two pointers",
}

func (a *App) tagsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List all known Codeforces problem tags",
		RunE: func(cmd *cobra.Command, _ []string) error {
			tags := make([]codeforces.Tag, len(knownTags))
			for i, t := range knownTags {
				tags[i] = codeforces.Tag{Name: t}
			}
			return a.render(tags)
		},
	}
	return cmd
}
