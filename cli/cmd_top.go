package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// topHandles is a curated list of legendary grandmaster handles used by cf top.
var topHandles = []string{
	"tourist", "Um_nik", "Radewoosh", "scott_wu", "Petr", "ksun48", "ecnerwala",
	"Benq", "jiangly", "maroonrk", "yosupo", "heno239", "orzdevinwang",
	"hos.lyric", "gamegame", "ko_osaga", "Ormlis", "neal", "SecondThread",
	"rainboy", "244mhq", "mnbvmar", "Retired_MiFaFaOvO", "xuanquang1999",
	"ffao", "HIR180", "dorijanlendvaj", "dario2994", "jqdai0815",
	"abc864197532", "darnley", "PavelKunyavskiy", "Merkurev", "krijgertje",
	"Um_nik",
}

func (a *App) topCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Show top-rated Codeforces users",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)

			// deduplicate handles
			seen := make(map[string]bool)
			var handles []string
			for _, h := range topHandles {
				lower := strings.ToLower(h)
				if !seen[lower] {
					seen[lower] = true
					handles = append(handles, h)
				}
			}
			if n > 0 && n < len(handles) {
				handles = handles[:n]
			}

			a.progressf("fetching top %d users...", len(handles))
			users, err := a.client.UserInfo(cmd.Context(), handles)
			if err != nil {
				return mapFetchErr(err)
			}

			sort.Slice(users, func(i, j int) bool {
				return users[i].Rating > users[j].Rating
			})

			return a.renderOrEmpty(users, len(users))
		},
	}
	return cmd
}
