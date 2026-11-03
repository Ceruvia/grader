package tasks

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"testing"
)

func TestTeacherArchiveRejectsAlternateZip64Directory(t *testing.T) {
	data := manyFlatFilesArchive(t)
	originalEnd := data[len(data)-22:]
	zip64Offset := len(data) - 22
	footer64 := make([]byte, 56)
	binary.LittleEndian.PutUint32(footer64[0:4], 0x06064b50)
	binary.LittleEndian.PutUint64(footer64[4:12], 44)
	binary.LittleEndian.PutUint16(footer64[12:14], 45)
	binary.LittleEndian.PutUint16(footer64[14:16], 45)
	binary.LittleEndian.PutUint64(footer64[24:32], 50000)
	binary.LittleEndian.PutUint64(footer64[32:40], 50000)
	binary.LittleEndian.PutUint64(footer64[40:48], uint64(binary.LittleEndian.Uint32(originalEnd[12:16])))
	binary.LittleEndian.PutUint64(footer64[48:56], uint64(binary.LittleEndian.Uint32(originalEnd[16:20])))
	data = append(bytes.Clone(data[:zip64Offset]), footer64...)
	classicOffset := len(data)
	// One harmless CEN record occupies exactly 65535 bytes. Its comment ends in
	// a locator naming the earlier ZIP64 directory with 50000 actual entries.
	classic := make([]byte, 65535)
	binary.LittleEndian.PutUint32(classic[0:4], 0x02014b50)
	binary.LittleEndian.PutUint16(classic[28:30], 1)
	binary.LittleEndian.PutUint16(classic[32:34], uint16(len(classic)-47))
	classic[46] = 'f'
	locator := classic[len(classic)-20:]
	binary.LittleEndian.PutUint32(locator[0:4], 0x07064b50)
	binary.LittleEndian.PutUint64(locator[8:16], uint64(zip64Offset))
	binary.LittleEndian.PutUint32(locator[16:20], 1)
	data = append(data, classic...)
	end := make([]byte, 22)
	binary.LittleEndian.PutUint32(end[0:4], 0x06054b50)
	binary.LittleEndian.PutUint16(end[8:10], 1)
	binary.LittleEndian.PutUint16(end[10:12], 1)
	binary.LittleEndian.PutUint32(end[12:16], 65535)
	binary.LittleEndian.PutUint32(end[16:20], uint32(classicOffset))
	data = append(data, end...)

	// This is a real parser differential, not a forged advertised count alone.
	// Go 1.26 selects ZIP64 on size 65535. Go 1.27 selects the classic directory.
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("counterexample must parse with the deployed reader: %v", err)
	}
	if len(reader.File) != 50000 && len(reader.File) != 1 {
		t.Fatalf("unexpected downstream directory selection: %d", len(reader.File))
	}
	t.Logf("downstream archive/zip selected %d actual CEN entries", len(reader.File))
	measured := &measuredReaderAt{ReaderAt: bytes.NewReader(data)}
	if err := checkTeacherZipDirectory(measured, int64(len(data))); err == nil {
		t.Fatalf("preflight accepted an ambiguous directory while archive/zip selected %d entries", len(reader.File))
	} else {
		t.Logf("preflight rejected before NewReader: %v, read %d bytes", err, measured.bytes)
	}
	if measured.bytes > 1<<20 {
		t.Fatalf("ambiguous framing must reject within the 1 MiB parsing budget, read %d bytes", measured.bytes)
	}

	// A classic directory of the same size without a locator is unambiguous.
	clear(data[len(data)-42 : len(data)-22])
	if err := checkTeacherZipDirectory(bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("ordinary 65535-byte classic directory was rejected: %v", err)
	}
	reader, err = zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(reader.File) != 1 {
		t.Fatalf("ordinary classic directory is not preserved: %v", err)
	}
}
