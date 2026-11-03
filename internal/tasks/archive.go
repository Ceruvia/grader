package tasks

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	teacherArchiveMaxBytes = 256 << 20
	teacherArchiveMaxFiles = 10000
)

func extractTeacherArchive(ctx context.Context, archivePath, destination string) error {
	file, err := os.Open(archivePath) //nolint:gosec // archivePath is grader.zip in the host-only TaskDir, built from the box id.
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if err := checkTeacherZipDirectory(file, info.Size()); err != nil {
		return err
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return err
	}
	var expanded uint64
	seen := make(map[string]bool, len(archive.File))
	for _, entry := range archive.File {
		name := filepath.Clean(entry.Name)
		if !filepath.IsLocal(name) || name == "." || strings.Contains(entry.Name, "\\") {
			return fmt.Errorf("unsafe teacher archive path %q", entry.Name)
		}
		if !entry.Mode().IsRegular() && !entry.Mode().IsDir() {
			return fmt.Errorf("teacher archive entry %q is not a regular file or directory", entry.Name)
		}
		if seen[name] {
			return fmt.Errorf("duplicate teacher archive path %q", entry.Name)
		}
		seen[name] = true
		if entry.UncompressedSize64 > teacherArchiveMaxBytes-expanded {
			return errors.New("teacher archive exceeds 256 MiB expanded size")
		}
		expanded += entry.UncompressedSize64
	}
	// Validate the whole archive before creating even its first destination file.
	for _, entry := range archive.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		target := filepath.Join(destination, entry.Name) //nolint:gosec // every name passed filepath.IsLocal above, and only regular files and dirs extract.
		if entry.Mode().IsDir() {
			if err := os.MkdirAll(target, 0700); err != nil {
				return err
			}
			continue
		}
		// 0700: only the grader process reads these. Files reach the box through CopyFile.
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err := extractTeacherEntry(entry, target); err != nil {
			return err
		}
	}
	// An empty archive still has a valid, empty extraction directory.
	return os.MkdirAll(destination, 0700)
}

func extractTeacherEntry(entry *zip.File, target string) error {
	source, err := entry.Open()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, entry.Mode().Perm()) //nolint:gosec // target is a validated local name under destination, O_EXCL.
	if err != nil {
		_ = source.Close()
		return err
	}
	// Enforce the declared size during expansion too, rather than trusting only
	// central-directory metadata. zip's reader also validates the CRC.
	n, copyErr := io.Copy(out, io.LimitReader(source, int64(entry.UncompressedSize64)+1))
	err = errors.Join(copyErr, source.Close(), out.Close())
	if err != nil {
		return err
	}
	if uint64(n) != entry.UncompressedSize64 {
		return errors.New("teacher archive entry size differs from metadata")
	}
	return nil
}
