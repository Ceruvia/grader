package tasks

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func manyFlatFilesArchive(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	for i := range 50000 {
		if _, err := writer.CreateHeader(&zip.FileHeader{Name: fmt.Sprintf("f%05d", i), Method: zip.Store}); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	result := data.Bytes()
	// The advertised count is intentionally false. Resource enforcement must
	// count actual central-directory records before archive/zip materializes them.
	end := result[len(result)-22:]
	binary.LittleEndian.PutUint16(end[8:10], 1)
	binary.LittleEndian.PutUint16(end[10:12], 1)
	return result
}

func TestTeacherArchiveBoundsDirectoryParsing(t *testing.T) {
	data := manyFlatFilesArchive(t)
	path := filepath.Join(t.TempDir(), "many-flat-files.zip")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "files")
	err := extractTeacherArchive(context.Background(), path, destination)
	if err == nil || !strings.Contains(err.Error(), "10000 entries") {
		t.Fatalf("must enforce actual entry limit before eager ZIP parsing, got %v", err)
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatalf("oversized directory must not extract files: %v", err)
	}
	measured := &measuredReaderAt{ReaderAt: bytes.NewReader(data)}
	if err := checkTeacherZipDirectory(measured, int64(len(data))); err != errTeacherArchiveEntryLimit {
		t.Fatalf("forged entry count bypassed the bounded parser: %v", err)
	}
	if measured.bytes > 1<<20 {
		t.Fatalf("parsing 50000 flat entries must stop within a 1 MiB read budget, read %d bytes", measured.bytes)
	}
}

type measuredReaderAt struct {
	io.ReaderAt
	bytes int
}

func (r *measuredReaderAt) ReadAt(p []byte, offset int64) (int, error) {
	n, err := r.ReaderAt.ReadAt(p, offset)
	r.bytes += n
	return n, err
}

func TestTeacherArchiveDirectoryFraming(t *testing.T) {
	var original bytes.Buffer
	writer := zip.NewWriter(&original)
	entry, err := writer.Create("support.h")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, "#define ANSWER 0\n"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	plain := original.Bytes()
	zip64 := teacherZip64(plain)
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"ordinary ZIP", plain},
		{"ZIP64 footer", zip64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "teacher.zip")
			if err := os.WriteFile(path, tc.data, 0600); err != nil {
				t.Fatal(err)
			}
			destination := filepath.Join(t.TempDir(), "files")
			if err := extractTeacherArchive(context.Background(), path, destination); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(filepath.Join(destination, "support.h"))
			if err != nil || string(body) != "#define ANSWER 0\n" {
				t.Fatalf("valid teacher archive changed: %q %v", body, err)
			}
		})
	}
	for _, tc := range []struct {
		name   string
		data   []byte
		mutate func([]byte)
	}{
		{"directory size mismatch", plain, func(data []byte) { binary.LittleEndian.PutUint32(data[len(data)-10:len(data)-6], 0xffffffff-1) }},
		{"ZIP64 offset overflow", zip64, func(data []byte) { binary.LittleEndian.PutUint64(data[len(data)-34:len(data)-26], ^uint64(0)) }},
		{"ZIP64 count overflow", zip64, func(data []byte) { binary.LittleEndian.PutUint64(data[len(data)-66:len(data)-58], ^uint64(0)) }},
		{"ZIP64 truncated footer", zip64, func(data []byte) { binary.LittleEndian.PutUint64(data[len(data)-94:len(data)-86], 43) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := bytes.Clone(tc.data)
			tc.mutate(data)
			if err := checkTeacherZipDirectory(bytes.NewReader(data), int64(len(data))); err == nil {
				t.Fatal("malformed directory framing was accepted")
			}
		})
	}
	t.Run("metadata byte bound", func(t *testing.T) {
		data := make([]byte, (9<<20)+22)
		end := data[len(data)-22:]
		binary.LittleEndian.PutUint32(end[0:4], 0x06054b50)
		binary.LittleEndian.PutUint16(end[8:10], 1)
		binary.LittleEndian.PutUint16(end[10:12], 1)
		binary.LittleEndian.PutUint32(end[12:16], 9<<20)
		if err := checkTeacherZipDirectory(bytes.NewReader(data), int64(len(data))); err == nil || !strings.Contains(err.Error(), "metadata exceeds 8 MiB") {
			t.Fatalf("metadata bytes were not bounded before CEN parsing: %v", err)
		}
	})
}

// ZIP APPNOTE's 56-byte ZIP64 end record and 20-byte locator, with classic
// footer fields replaced by the ZIP64 sentinel values.
func teacherZip64(plain []byte) []byte {
	endOffset := len(plain) - 22
	end := bytes.Clone(plain[endOffset:])
	footer := make([]byte, 56)
	binary.LittleEndian.PutUint32(footer[0:4], 0x06064b50)
	binary.LittleEndian.PutUint64(footer[4:12], 44)
	binary.LittleEndian.PutUint16(footer[12:14], 45)
	binary.LittleEndian.PutUint16(footer[14:16], 45)
	binary.LittleEndian.PutUint64(footer[24:32], uint64(binary.LittleEndian.Uint16(end[8:10])))
	binary.LittleEndian.PutUint64(footer[32:40], uint64(binary.LittleEndian.Uint16(end[10:12])))
	binary.LittleEndian.PutUint64(footer[40:48], uint64(binary.LittleEndian.Uint32(end[12:16])))
	binary.LittleEndian.PutUint64(footer[48:56], uint64(binary.LittleEndian.Uint32(end[16:20])))
	locator := make([]byte, 20)
	binary.LittleEndian.PutUint32(locator[0:4], 0x07064b50)
	binary.LittleEndian.PutUint64(locator[8:16], uint64(endOffset))
	binary.LittleEndian.PutUint32(locator[16:20], 1)
	binary.LittleEndian.PutUint16(end[8:10], 0xffff)
	binary.LittleEndian.PutUint16(end[10:12], 0xffff)
	binary.LittleEndian.PutUint32(end[12:16], 0xffffffff)
	binary.LittleEndian.PutUint32(end[16:20], 0xffffffff)
	result := bytes.Clone(plain[:endOffset])
	result = append(result, footer...)
	result = append(result, locator...)
	return append(result, end...)
}
