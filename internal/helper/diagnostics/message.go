package diagnostics

import (
	"strings"
	"unicode/utf8"
)

const MaxBytes = 64 << 10

// Limit bounds the decoded JSON message, including replacement of invalid UTF-8.
func Limit(message string) string { return Prefix(message, MaxBytes) }

func Prefix(message string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	message = strings.ToValidUTF8(message, "\ufffd")
	if len(message) <= maxBytes {
		return message
	}
	end := maxBytes
	for !utf8.RuneStart(message[end]) {
		end--
	}
	return message[:end]
}

// Buffer drains subprocess output while retaining only the diagnostic prefix.
type Buffer struct {
	data []byte
}

func (b *Buffer) Write(p []byte) (int, error) {
	n := len(p)
	if remaining := MaxBytes - len(b.data); remaining > 0 {
		b.data = append(b.data, p[:min(n, remaining)]...)
	}
	return n, nil
}

func (b *Buffer) String() string { return Limit(string(b.data)) }
