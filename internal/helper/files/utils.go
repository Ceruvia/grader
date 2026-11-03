package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// CopyFile copies a grader-owned host file into the box. dst sits in the box, where
// earlier student code may have planted a symlink or FIFO: it is replaced, never followed.
func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src) //nolint:gosec // src is a teacher file in the host-only task dir, never a student path.
	if err != nil {
		return 0, err
	}
	defer source.Close()

	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	// 0666 like os.Create, before umask: the unprivileged box uid must read the file.
	destination, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0666) //nolint:gosec // O_EXCL|O_NOFOLLOW never write through a planted link.
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func RemoveExtention(filename string) string {
	extention := filepath.Ext(filename)
	return strings.TrimSuffix(filename, extention)
}

func GetFilenamesInDir(dirpath string) ([]string, error) {
	entries, err := os.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}

	var filenames []string
	for _, entry := range entries {
		filenames = append(filenames, entry.Name())
	}
	return filenames, nil
}
