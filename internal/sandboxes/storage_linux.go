//go:build linux

package sandboxes

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// isolate is installed with PREFIX=/usr/local in the grader image. Its root is
// read from the same configuration as the executable, never from a box file.
var isolateBoxRoot = sync.OnceValues(func() (string, error) {
	file, err := os.Open("/usr/local/etc/isolate")
	if err != nil {
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if ok && strings.TrimSpace(key) == "box_root" {
			root := strings.TrimSpace(value)
			if !filepath.IsAbs(root) || filepath.Clean(root) == "/" {
				return "", errors.New("invalid isolate box_root")
			}
			return root, nil
		}
	}
	return "", errors.Join(errors.New("isolate box_root not configured"), scanner.Err())
})

func recoverBoxStorage(boxID int) error {
	root, err := isolateBoxRoot()
	if err != nil {
		return fmt.Errorf("read isolate box_root: %w", err)
	}
	// A process restart does not remove mounts. Detach our known mount before
	// --init recursively removes the old box, including partially initialized ones.
	return unmountBoxStorage(filepath.Join(root, strconv.Itoa(boxID), "box"))
}

// A separate tmpfs per box enforces aggregate byte and inode limits even when
// the parent overlay filesystem cannot support isolate's filesystem quotas.
func mountBoxStorage(path string) error {
	return unix.Mount("tmpfs", path, "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, "size=268435456,nr_inodes=10000,mode=0777")
}

// onUnmountBusy runs after each EBUSY. Tests use it to release a holder only
// once a real unmount attempt has failed.
var onUnmountBusy = func() {}

// unmountBoxStorage waits out short EBUSY. A fork in another worker (isolate for
// a different box) copies every grader fd, including a box file the grader is
// about to close, and holds it until the child execs. A holder that outlives the
// bound still fails, so the pool quarantines the box instead of reusing it dirty.
func unmountBoxStorage(path string) error {
	deadline := time.Now().Add(2 * time.Second)
	for wait := time.Millisecond; ; wait = min(2*wait, 50*time.Millisecond) {
		err := unix.Unmount(path, 0)
		switch {
		case err == nil, errors.Is(err, unix.EINVAL), errors.Is(err, unix.ENOENT):
			return nil
		case !errors.Is(err, unix.EBUSY) || !time.Now().Before(deadline):
			return fmt.Errorf("unmount bounded box storage: %w", err)
		}
		onUnmountBusy()
		time.Sleep(min(wait, time.Until(deadline)))
	}
}
