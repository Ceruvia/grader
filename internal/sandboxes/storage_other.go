//go:build !linux

package sandboxes

import "errors"

func mountBoxStorage(string) error {
	return errors.New("bounded sandbox storage requires Linux")
}

func unmountBoxStorage(string) error { return nil }
