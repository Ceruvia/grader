package sandboxes

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

var (
	ErrNotRegularFile  = errors.New("box file is not a regular file")
	ErrBoxFileTooLarge = errors.New("box file exceeds the box file size limit")
)

// defaultReadLimitBytes applies when a sandbox has no file size limit. It equals the
// CreateIsolateSandbox default of 102400 KB.
const defaultReadLimitBytes = 100 << 20

// ReadBoxFile reads a file the sandboxed program could have written. The grader runs as
// root, so it never follows a symlink, never blocks on a FIFO or device, and never reads
// more than the box file size limit (isolate -f, in KB), which bounds any file the program
// can write.
func ReadBoxFile(sbx Sandbox, filename string) ([]byte, error) {
	f, err := os.OpenFile(filepath.Join(sbx.GetBoxdir(), filename), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: %w", filename, ErrNotRegularFile)
	}

	limit := int64(defaultReadLimitBytes)
	if kb := sbx.GetFileSizeLimit(); kb > 0 {
		limit = int64(kb) * 1024
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s: %w", filename, ErrBoxFileTooLarge)
	}
	return data, nil
}
