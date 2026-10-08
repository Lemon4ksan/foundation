package matchfinder

import "fmt"

// A TextEncoder is an Encoder that produces a human-readable representation of
// the LZ77 compression. Matches are replaced with <Length,Distance> symbols.
type TextEncoder struct{}

func (t TextEncoder) Reset() { return }

func (t TextEncoder) Encode(dst, src []byte, matches []Match, lastBlock bool) []byte {
	pos := 0
	for _, m := range matches {
		if m.Unmatched > 0 {
			dst = append(dst, src[pos:pos+m.Unmatched]...)
			pos += m.Unmatched
		}
		if m.Length > 0 {
			dst = append(dst, []byte(fmt.Sprintf("<%d,%d>", m.Length, m.Distance))...)
			pos += m.Length
		}
	}
	if pos < len(src) {
		dst = append(dst, src[pos:]...)
	}
	return dst
}

// A NoFinder implements Finder, but doesn't find any matches.
// It can be used to implement the equivalent of the standard library flate package's
// HuffmanOnly setting.
type NoFinder struct{}

func (n NoFinder) Reset() { return }

func (n NoFinder) FindMatches(dst []Match, src []byte) []Match {
	return append(dst, Match{
		Unmatched: len(src),
	})
}

// AutoReset wraps a Finder that can return references to data in previous
// blocks, and calls Reset before each block. It is useful for (e.g.) using a
// snappy Encoder with a Finder designed for flate. (Snappy doesn't
// support references between blocks.)
type AutoReset struct {
	Finder
}

func (a AutoReset) FindMatches(dst []Match, src []byte) []Match {
	a.Reset()
	return a.Finder.FindMatches(dst, src)
}
