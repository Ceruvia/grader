package files

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func MoveFileFromDirToDir(srcDir, dstDir string) error {
	info, err := os.Stat(dstDir)
	if os.IsNotExist(err) || !info.IsDir() {
		return err
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Type().IsRegular() {
			srcPath := filepath.Join(srcDir, entry.Name())
			dstPath := filepath.Join(dstDir, entry.Name())

			if err := MoveFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func MoveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	if err := os.Remove(src); err != nil {
		return err
	}

	return nil
}

func RemoveExtention(filename string) string {
	extention := filepath.Ext(filename)
	return strings.TrimSuffix(filename, extention)
}
