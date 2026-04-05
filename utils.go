package main

import (
	"crypto/sha256"
	"encoding/binary"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

func SplitWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading spaces.
	start := 0
	for start < len(data) {
		r, width := utf8.DecodeRune(data[start:])
		if !unicode.IsSpace(r) {
			break
		}
		start += width
	}
	// Scan until whitespace, marking end of word.
	curr := start
	punctWidth := 0
	for curr < len(data) {
		r, width := utf8.DecodeRune(data[curr:])
		if unicode.IsSpace(r) && curr > start {
			// Return current word excluding end punctuation
			return curr - punctWidth, data[start : curr-punctWidth], nil
		}
		if isPunctuation(r) {
			if curr == start {
				// If punctuation at start of word, return it alone
				return curr + width, data[start : curr+width], nil
			} else {
				// If punctuation in between word, exclude from returned token
				punctWidth += width
			}
		} else {
			punctWidth = 0
		}
		curr += width
	}
	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && curr > start {
		return curr, data[start:curr], nil
	}
	// Request more data.
	return start, nil, nil
}

func isPunctuation(r rune) bool {
	return !(unicode.IsSpace(r) || unicode.IsLetter(r) || unicode.IsNumber(r))
}

func isOpeningPunct(r rune) bool {
	return unicode.In(r, unicode.Ps, unicode.Pi)
}

func isClosingPunct(r rune) bool {
	return unicode.In(r, unicode.Pe, unicode.Po, unicode.Pf)
}

func StringHasher(s []string) string {
	return strings.Join(s, " ")
}

func StringHasherConst3(s []string) [3]string {
	if len(s) != 3 {
		panic("unknown len")
	}
	return [3]string{s[0], s[1], s[2]}
}

func Int64FromBytes(data []byte) int64 {
	hash := sha256.Sum256(data)
	seed := int64(binary.BigEndian.Uint64(hash[:8]))
	return seed
}

type CountingWriter struct {
	w io.Writer
	C int
}

var _ io.Writer = &CountingWriter{}

func NewCountingWriter(w io.Writer) *CountingWriter {
	return &CountingWriter{
		w: w,
		C: 0,
	}
}

func (w *CountingWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.C += n
	return n, err
}
