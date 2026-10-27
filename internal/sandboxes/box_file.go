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

// OpenBoxFile opens untrusted output without following symlinks or blocking on
// special files. The returned descriptor belongs to the caller.
func OpenBoxFile(sbx Sandbox, filename string) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(sbx.GetBoxdir(), filename), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*os.File, error) {
		_ = f.Close()
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		return fail(err)
	}
	if !info.Mode().IsRegular() {
		return fail(fmt.Errorf("%s: %w", filename, ErrNotRegularFile))
	}
	if info.Size() > boxFileLimit(sbx) {
		return fail(fmt.Errorf("%s: %w", filename, ErrBoxFileTooLarge))
	}
	return f, nil
}

func boxFileLimit(sbx Sandbox) int64 {

	limit := int64(defaultReadLimitBytes)
	if kb := sbx.GetFileSizeLimit(); kb > 0 {
		limit = int64(kb) * 1024
	}
	return limit
}

// ReadBoxFile reads only the requested diagnostic prefix. Zero retains the full
// file API, bounded by the sandbox's per-file limit.
func ReadBoxFile(sbx Sandbox, filename string, maxBytes int64) ([]byte, error) {
	f, err := OpenBoxFile(sbx, filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	limit := boxFileLimit(sbx)
	if maxBytes > 0 && maxBytes < limit {
		limit = maxBytes
	}
	return io.ReadAll(io.LimitReader(f, limit))
}
