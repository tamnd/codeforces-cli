package cli

import (
	"sort"

	"github.com/spf13/cobra"
	"github.com/tamnd/codeforces-cli/codeforces"
)

func (a *App) contestsCmd() *cobra.Command {
	var upcoming bool
	cmd := &cobra.Command{
		Use:   "contests",
		Short: "List Codeforces contests",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching contests...")
			contests, err := a.client.Contests(cmd.Context(), false)
			if err != nil {
				return mapFetchErr(err)
			}

			if upcoming {
				var before []codeforces.Contest
				for _, c := range contests {
					if c.Phase == "BEFORE" {
						before = append(before, c)
					}
				}
				// sort upcoming ascending by start (soonest first = rank 1)
				sort.Slice(before, func(i, j int) bool {
					return before[i].Start < before[j].Start
				})
				contests = before
			} else {
				// sort all by start descending (most recent first)
				sort.Slice(contests, func(i, j int) bool {
					return contests[i].Start > contests[j].Start
				})
			}

			if n > 0 && n < len(contests) {
				contests = contests[:n]
			}
			for i := range contests {
				contests[i].Rank = i + 1
			}
			return a.renderOrEmpty(contests, len(contests))
		},
	}
	cmd.Flags().BoolVar(&upcoming, "upcoming", false, "show only upcoming contests (phase=BEFORE)")
	return cmd
}
