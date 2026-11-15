//go:build !linux && !darwin

package build

import "errors"

func exchangeDirectories(a, b string) error {
	return errors.New("atomic catalog replacement is unsupported on this host; existing catalog preserved")
}
