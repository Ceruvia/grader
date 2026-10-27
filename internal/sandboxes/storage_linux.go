//go:build linux

package sandboxes

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// A separate tmpfs per box enforces aggregate byte and inode limits even when
// the parent overlay filesystem cannot support isolate's filesystem quotas.
func mountBoxStorage(path string) error {
	return unix.Mount("tmpfs", path, "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, "size=268435456,nr_inodes=10000,mode=0777")
}

func unmountBoxStorage(path string) error {
	err := unix.Unmount(path, 0)
	if errors.Is(err, unix.EINVAL) || errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("unmount bounded box storage: %w", err)
	}
	return nil
}
