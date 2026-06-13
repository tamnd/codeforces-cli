package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) problemsCmd() *cobra.Command {
	var (
		tag       string
		minRating int
		maxRating int
	)
	cmd := &cobra.Command{
		Use:   "problems",
		Short: "List Codeforces problems",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching problems...")
			probs, err := a.client.Problems(cmd.Context(), tag, minRating, maxRating)
			if err != nil {
				return mapFetchErr(err)
			}
			if n > 0 && n < len(probs) {
				probs = probs[:n]
			}
			return a.renderOrEmpty(probs, len(probs))
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag (e.g. dp, greedy, math)")
	cmd.Flags().IntVar(&minRating, "min-rating", 0, "minimum problem rating (0 = no bound)")
	cmd.Flags().IntVar(&maxRating, "max-rating", 0, "maximum problem rating (0 = no bound)")
	return cmd
}
