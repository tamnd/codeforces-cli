package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/codeforces-cli/codeforces"
)

func (a *App) tagsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List all known Codeforces problem tags",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.render(codeforces.Tags())
		},
	}
	return cmd
}
