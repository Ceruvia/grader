package tasks

import (
	"archive/zip"
	"bufio"
	"encoding/binary"
	"errors"
	"io"

	"github.com/Ceruvia/grader/internal/helper/files"
)

const teacherArchiveMaxDirectoryBytes = 8 << 20

var errTeacherArchiveEntryLimit = errors.New("teacher archive exceeds 10000 entries")

// Bound metadata BEFORE archive/zip eagerly allocates one File per directory
// record. Only ZIP framing is inspected here. The standard reader still owns
// decompression, size validation, ZIP64 entry details and CRC integrity checks.
func checkTeacherZipDirectory(source io.ReaderAt, size int64) error {
	if size < 22 || size > files.MaxDownloadBytes {
		return zip.ErrFormat
	}
	tail := make([]byte, min(size, 22+65535))
	if _, err := source.ReadAt(tail, size-int64(len(tail))); err != nil {
		return err
	}
	end := -1
	for i := len(tail) - 22; i >= 0; i-- {
		if binary.LittleEndian.Uint32(tail[i:i+4]) == 0x06054b50 {
			// Reject ambiguous/trailing metadata rather than letting our preflight
			// and archive/zip select different end records inside a ZIP comment.
			if int(binary.LittleEndian.Uint16(tail[i+20:i+22])) != len(tail)-i-22 {
				return zip.ErrFormat
			}
			end = i
			break
		}
	}
	if end < 0 {
		return zip.ErrFormat
	}
	footer := tail[end:]
	if binary.LittleEndian.Uint16(footer[4:6]) != 0 || binary.LittleEndian.Uint16(footer[6:8]) != 0 {
		return zip.ErrFormat
	}
	records := uint64(binary.LittleEndian.Uint16(footer[10:12]))
	diskRecords := uint64(binary.LittleEndian.Uint16(footer[8:10]))
	directorySize := uint64(binary.LittleEndian.Uint32(footer[12:16]))
	directoryOffset := uint64(binary.LittleEndian.Uint32(footer[16:20]))
	directoryEnd := uint64(size - int64(len(tail)) + int64(end))
	if directorySize == 0xffff && records != 0xffff && directoryOffset != 0xffffffff && directoryEnd >= 20 {
		// Go 1.26 treats size=65535 as a ZIP64 selector, while newer readers
		// treat it as a classic size. A locator hidden in the classic CEN
		// comment can redirect only one reader to an entirely different
		// directory. Reject that ambiguity before either eager reader runs.
		var locator [20]byte
		if _, err := source.ReadAt(locator[:], int64(directoryEnd)-20); err != nil {
			return err
		}
		if binary.LittleEndian.Uint32(locator[0:4]) == 0x07064b50 {
			return errors.New("ambiguous teacher ZIP64 directory selection")
		}
	}
	if records == 0xffff || directorySize == 0xffffffff || directoryOffset == 0xffffffff {
		var locator [20]byte
		if directoryEnd < uint64(len(locator)) {
			return zip.ErrFormat
		}
		locatorOffset := directoryEnd - uint64(len(locator))
		if _, err := source.ReadAt(locator[:], int64(locatorOffset)); err != nil {
			return err
		}
		if binary.LittleEndian.Uint32(locator[0:4]) != 0x07064b50 || binary.LittleEndian.Uint32(locator[4:8]) != 0 || binary.LittleEndian.Uint32(locator[16:20]) != 1 {
			return zip.ErrFormat
		}
		zip64Offset := binary.LittleEndian.Uint64(locator[8:16])
		var footer64 [56]byte
		if zip64Offset > locatorOffset || locatorOffset-zip64Offset < uint64(len(footer64)) {
			return zip.ErrFormat
		}
		if _, err := source.ReadAt(footer64[:], int64(zip64Offset)); err != nil {
			return err
		}
		if binary.LittleEndian.Uint32(footer64[0:4]) != 0x06064b50 || binary.LittleEndian.Uint32(footer64[16:20]) != 0 || binary.LittleEndian.Uint32(footer64[20:24]) != 0 {
			return zip.ErrFormat
		}
		if binary.LittleEndian.Uint64(footer64[4:12]) != locatorOffset-zip64Offset-12 {
			return zip.ErrFormat
		}
		diskRecords = binary.LittleEndian.Uint64(footer64[24:32])
		records = binary.LittleEndian.Uint64(footer64[32:40])
		directorySize = binary.LittleEndian.Uint64(footer64[40:48])
		directoryOffset = binary.LittleEndian.Uint64(footer64[48:56])
		directoryEnd = zip64Offset
	}
	if records > teacherArchiveMaxFiles {
		return errTeacherArchiveEntryLimit
	}
	if diskRecords != records || directoryOffset > directoryEnd || directorySize != directoryEnd-directoryOffset {
		return zip.ErrFormat
	}
	if directorySize > teacherArchiveMaxDirectoryBytes {
		return errors.New("teacher archive metadata exceeds 8 MiB")
	}

	// Do not trust the advertised count: count actual CEN records using a fixed
	// buffer and stop at the policy limit, even when an attacker forges EOCD.
	directory := bufio.NewReader(io.NewSectionReader(source, int64(directoryOffset), int64(directorySize)))
	var header [46]byte
	var count uint64
	for remaining := directorySize; remaining > 0; {
		if count == teacherArchiveMaxFiles {
			return errTeacherArchiveEntryLimit
		}
		if remaining < uint64(len(header)) {
			return zip.ErrFormat
		}
		if _, err := io.ReadFull(directory, header[:]); err != nil {
			return err
		}
		if binary.LittleEndian.Uint32(header[0:4]) != 0x02014b50 || binary.LittleEndian.Uint16(header[34:36]) != 0 {
			return zip.ErrFormat
		}
		variable := int(binary.LittleEndian.Uint16(header[28:30])) + int(binary.LittleEndian.Uint16(header[30:32])) + int(binary.LittleEndian.Uint16(header[32:34]))
		remaining -= uint64(len(header))
		if uint64(variable) > remaining {
			return zip.ErrFormat
		}
		if _, err := directory.Discard(variable); err != nil {
			return err
		}
		remaining -= uint64(variable)
		count++
	}
	if count != records {
		return zip.ErrFormat
	}
	return nil
}
