package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) userCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user <handle>",
		Short: "Show a Codeforces user profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handle := args[0]
			a.progressf("fetching user %q...", handle)
			users, err := a.client.UserInfo(cmd.Context(), []string{handle})
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(users, len(users))
		},
	}
	return cmd
}
