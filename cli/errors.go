package cli

import (
	"errors"

	"github.com/tamnd/codeforces-cli/codeforces"
)

func isNotFound(err error) bool {
	return errors.Is(err, codeforces.ErrNotFound)
}
