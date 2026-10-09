package lz4block

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestBlockDecode(t *testing.T) {
	appendLen := func(p []byte, size int) []byte {
		for size > 0xFF {
			p = append(p, 0xFF)
			size -= 0xFF
		}

		p = append(p, byte(size))
		return p
	}

	emitSeq := func(lit string, offset uint16, matchLen int) []byte {
		var b byte
		litLen := len(lit)
		if litLen < 15 {
			b = byte(litLen << 4)
			litLen = -1
		} else {
			b = 0xF0
			litLen -= 15
		}

		if matchLen < 4 || offset == 0 {
			out := []byte{b}
			if litLen >= 0 {
				out = appendLen(out, litLen)
			}
			return append(out, lit...)
		}

		matchLen -= 4
		if matchLen < 15 {
			b |= byte(matchLen)
			matchLen = -1
		} else {
			b |= 0x0F
			matchLen -= 15
		}

		out := []byte{b}
		if litLen >= 0 {
			out = appendLen(out, litLen)
		}

		if len(lit) > 0 {
			out = append(out, lit...)
		}

		out = append(out, byte(offset), byte(offset>>8))

		if matchLen >= 0 {
			out = appendLen(out, matchLen)
		}

		return out
	}
	concat := func(in ...[]byte) []byte {
		var p []byte
		for _, b := range in {
			p = append(p, b...)
		}
		return p
	}

	tests := []struct {
		name string
		src  []byte
		exp  []byte
	}{
		{
			"empty_input",
			[]byte{0},
			[]byte{},
		},
		{
			"empty_input_nil_dst",
			[]byte{0},
			nil,
		},
		{
			"literal_only_short",
			emitSeq("hello", 0, 0),
			[]byte("hello"),
		},
		{
			"literal_only_long",
			emitSeq(strings.Repeat("A", 15+255+255+1), 0, 0),
			bytes.Repeat([]byte("A"), 15+255+255+1),
		},
		{
			"literal_only_long_1",
			emitSeq(strings.Repeat("A", 15), 0, 0),
			bytes.Repeat([]byte("A"), 15),
		},
		{
			"literal_only_long_2",
			emitSeq(strings.Repeat("A", 16), 0, 0),
			bytes.Repeat([]byte("A"), 16),
		},
		{
			"repeat_match_len",
			emitSeq("a", 1, 4),
			[]byte("aaaaa"),
		},
		{
			"repeat_match_len_2_seq",
			concat(emitSeq("a", 1, 4), emitSeq("B", 1, 4)),
			[]byte("aaaaaBBBBB"),
		},
		{
			"long_match",
			emitSeq("A", 1, 16),
			bytes.Repeat([]byte("A"), 17),
		},
		{
			// Triggers a case in the amd64 decoder where its last action
			// is a call to runtime.memmove.
			"memmove_last",
			emitSeq(strings.Repeat("ABCD", 20), 80, 60),
			bytes.Repeat([]byte("ABCD"), 36)[:140],
		},
		{
			"repeat_match_log_len_2_seq",
			concat(emitSeq("a", 1, 15), emitSeq("B", 1, 15), emitSeq("end", 0, 0)),
			[]byte(strings.Repeat("a", 16) + strings.Repeat("B", 16) + "end"),
		},
		{
			"fuzz-3e4ce8cc0da392ca5a353b6ffef6d08f400ac5f9",
			[]byte("0000\x01\x00"),
			[]byte("0000000"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := make([]byte, len(test.exp))
			n := decodeBlock(buf, test.src, nil)
			if n != len(test.exp) {
				t.Errorf("expected %d, got %d", len(test.exp), n)
			}
			if !bytes.Equal(buf, test.exp) {
				t.Fatalf("expected %q got %q", test.exp, buf)
			}
		})
	}
}

func TestDecodeBlockInvalid(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		src  string
		size int // Output size to try.
	}{
		{
			"empty_input",
			"",
			100,
		},
		{
			// A string of zeros could be interpreted as empty match+empty literal repeated,
			// but only the last block may have an empty match (with the offset missing).
			"repeated_zero_few",
			string(make([]byte, 6)),
			100,
		},
		{
			"repeated_zero_many",
			string(make([]byte, 100)),
			100,
		},
		{
			"final_lit_too_short",
			"\x20a", // litlen = 2 but only a single-byte literal
			100,
		},
		{
			"no_space_for_offset",
			"\x01", // mLen > 0 but no following offset.
			10,
		},
		{
			"write_beyond_len_dst",
			"\x1b0\x01\x00000000000000",
			len("\x1b0\x01\x00000000000000"),
		},
		{
			"bounds-crasher", // Triggered a broken bounds check in amd64 decoder.
			"\x000000",
			10,
		},
		{
			// Zero offset in a short literal.
			"zero_offset",
			"\xe1abcdefghijklmn\x00\x00\xe0abcdefghijklmn",
			40,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dst := make([]byte, test.size+8)
			for i := range dst {
				dst[i] = byte(i)
			}
			dst = dst[:test.size]

			r := decodeBlock(dst, []byte(test.src), nil)
			if r >= 0 {
				t.Errorf("no error for %s", test.name)
			}

			dst = dst[:cap(dst)]
			for i := test.size; i < len(dst); i++ {
				if dst[i] != byte(i) {
					t.Error("decodeBlock wrote out of bounds")
					break
				}
			}
		})
	}
}

// Literal lengths should be checked for overflow.
//
// This test exists primarily for 32-bit platforms. Since a length n takes
// around n/255 bytes to encode, overflow can only occur on a 64-bit processor
// in blocks larger than 32PiB and decodeBlock will error out because the
// literal is too small.
func TestLongLengths(t *testing.T) {
	// n + 15 is large enough to overflow uint32.
	const (
		n      = (1 << 32) / 255
		remain = (255*n + 15) % (1 << 32)
	)

	src := make([]byte, 0, 100+n) // Just over 16MiB.
	src = append(src, '\xf0')
	for range n {
		src = append(src, 255)
	}
	src = append(src, 0)
	for i := range remain {
		src = append(src, 'A'+byte(i))
	}

	dst := make([]byte, 2*remain)
	r := decodeBlock(dst, src, nil)
	if r >= 0 {
		t.Errorf("want error, got %d (remain=%d)", r, remain)
	}
}

func TestDecodeWithDict(t *testing.T) {
	for _, c := range []struct {
		src, dict, want string // If want == "", expect an error.
	}{
		// Entire match in dictionary.
		{"\x11b\x0a\x00\x401234", "barbazquux", "barbaz1234"},

		// First part in dictionary, rest in dst.
		{"\x35foo\x09\x00\x401234", "0barbaz", "foobarbazfoo1234"},

		// Copy end of dictionary three times, then a literal.
		{"\x08\x04\x00\x50abcde", "---1234", "123412341234abcde"},

		// First part in dictionary, rest in dst, copied multiple times.
		{"\x1a1\x05\x00\x50abcde", "---2345", "123451234512345abcde"},

		// First part in dictionary, rest in dst, but >16 bytes before the end,
		// to test the short match shortcut in the amd64 decoder.
		{
			"\x35abc\x09\x00\xf0\x0f0123456789abcdefghijklmnopqrst", "012defghi",
			"abcdefghiabc0123456789abcdefghijklmnopqrst",
		},

		// Offset points before start of dictionary.
		{"\x35foo\xff\xff\x401234", "XYZ", ""},
	} {
		// Add sentinel for debugging.
		dict := []byte(c.dict + "ABCD")[:len(c.dict)]
		dst := make([]byte, 100)

		r := decodeBlock(dst, []byte(c.src), dict)

		if c.want == "" {
			if r >= 0 {
				t.Error("expected an error, got", r)
			}
		} else {
			switch {
			case r < 0:
				t.Errorf("error %d for %q", r, c.want)
			case string(dst[:r]) != c.want:
				t.Errorf("expected %q, got %q", c.want, dst[:r])
			}
		}
	}
}

// These tests directly exercise the match-copy paths in decodeBlock with
// hand-assembled LZ4 blocks, so they fail loudly if a specific (offset,
// matchlen) combination regresses. They were written after an arm64
// implementation hit a silent CCMP immediate truncation that only corrupted
// output on specific offsets — TestCompressUncompressBlock caught it late
// (one failing byte far into a large input), while TestMatchCopyMatrix
// reproduces the failure on the very first out-of-range combination.

// buildSingleMatchBlock produces an LZ4 block consisting of:
//   - `prefix` bytes of deterministic literal content, and
//   - a single match of length `mlen` at back-reference distance `offset`.
//
// Returns the compressed block and the expected decompressed output.
func buildSingleMatchBlock(prefix, offset, mlen int) (compressed, decoded []byte) {
	if prefix < offset {
		panic("prefix must be >= offset so the match has valid source data")
	}
	if mlen < minMatch {
		panic("match length must be >= minMatch")
	}

	decoded = make([]byte, prefix+mlen)
	for i := range prefix {
		decoded[i] = byte(i%253 + 1) // non-zero so errors are obvious
	}
	for i := range mlen {
		decoded[prefix+i] = decoded[prefix+i-offset]
	}

	var buf bytes.Buffer
	writeToken(&buf, prefix, mlen-minMatch)
	buf.Write(decoded[:prefix])
	off := make([]byte, 2)
	binary.LittleEndian.PutUint16(off, uint16(offset))
	buf.Write(off)
	if rawM := mlen - minMatch; rawM >= 15 {
		writeExtended(&buf, rawM-15)
	}
	return buf.Bytes(), decoded
}

func writeToken(buf *bytes.Buffer, litlen, rawMatch int) {
	tokLit := min(litlen, 15)
	tokM := min(rawMatch, 15)
	buf.WriteByte(byte((tokLit << 4) | tokM))
	if litlen >= 15 {
		writeExtended(buf, litlen-15)
	}
}

func writeExtended(buf *bytes.Buffer, rem int) {
	for rem >= 255 {
		buf.WriteByte(0xFF)
		rem -= 255
	}
	buf.WriteByte(byte(rem))
}

// TestMatchCopySingle covers representative single-match cases spanning each
// of the fast paths (4-byte, 8-byte, and 16-byte) and each of the tricky
// cycling boundaries (offset == matchlen, offset == 8, etc.).
func TestMatchCopySingle(t *testing.T) {
	cases := []struct {
		name         string
		offset, mlen int
		prefix       int // defaults to offset when 0
	}{
		{"off1_len8_rle", 1, 8, 64},       // splat, byte RLE
		{"off1_len100_rle", 1, 100, 64},   // splat, long RLE
		{"off2_len16_rle", 2, 16, 64},     // splat halfword
		{"off3_len16_rle", 3, 16, 64},     // tile path, exactly one prefill
		{"off4_len8", 4, 8, 64},           // word-splat path
		{"off8_len8", 8, 8, 64},           // shortcut 8+8+2 boundary
		{"off8_len18", 8, 18, 64},         // shortcut + match-len boundary
		{"off8_len31", 8, 31, 64},         // copyMatchLoop8 at offset=8 (below tile threshold)
		{"off8_len32", 8, 32, 64},         // offset-8 tile: exactly the threshold
		{"off8_len100", 8, 100, 64},       // offset-8 tile with 8B + byte tail
		{"off16_len32", 16, 32, 64},       // offset-16 tile, no tail
		{"off16_len47", 16, 47, 64},       // offset-16 tile with 15-byte tail
		{"off12_len64", 12, 64, 64},       // 9..15 generic tile (prefill 12)
		{"off17_len32", 17, 32, 64},       // 32B tile: prefill 17 = 2x8 + 1, no tile iteration
		{"off24_len96", 24, 96, 64},       // 32B tile: prefill 3x8, three iterations
		{"off31_len4096", 31, 4096, 64},   // 32B tile: long
		{"off16_len16", 16, 16, 64},       // offset >= 16 but < 32
		{"off16_len100", 16, 100, 64},     // offset < 32: must use 8B loop
		{"off32_len16", 32, 16, 64},       // smallest 16B-eligible
		{"off32_len32", 32, 32, 64},       // two 16B iters
		{"off32_len100", 32, 100, 128},    // longer
		{"off64_len4096", 64, 4096, 8192}, // bulk
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prefix := c.prefix
			if prefix == 0 {
				prefix = c.offset
			}
			src, want := buildSingleMatchBlock(prefix, c.offset, c.mlen)
			dst := make([]byte, len(want))
			n := decodeBlock(dst, src, nil)
			if n != len(want) {
				t.Fatalf("decode returned %d, want %d", n, len(want))
			}
			if !bytes.Equal(dst[:n], want) {
				for i := range n {
					if dst[i] != want[i] {
						t.Fatalf("first mismatch at byte %d: got 0x%02x want 0x%02x",
							i, dst[i], want[i])
					}
				}
			}
		})
	}
}

// TestMatchCopyMatrix exhaustively sweeps (offset, matchlen) over the ranges
// most likely to expose addressing-mode or threshold bugs. Each combination
// is decoded and diffed against the naive reference built in memory.
func TestMatchCopyMatrix(t *testing.T) {
	// Offsets: cover 1..7 (splat/tile paths), every offset in 8..31 (the
	// 8-byte loop and the offset-8/16/9..15/17..31 tile paths, whose
	// prefill and tail handling depend on offset%8 and on the exact
	// remaining length), and 32+ (16B loop / memmove). Include the exact
	// threshold points plus a scattering above.
	offsets := []int{1, 2, 3, 4, 5, 6, 7}
	for o := 8; o <= 33; o++ {
		offsets = append(offsets, o)
	}
	offsets = append(offsets, 48, 64, 127, 128, 255, 1024)
	// Match lengths: every integer from 4..64 catches off-by-one issues in
	// all the loops; then a few large values exercise the bulk-copy path.
	var mlens []int
	for l := minMatch; l <= 64; l++ {
		mlens = append(mlens, l)
	}
	mlens = append(mlens, 65, 66, 71, 72, 79, 80, 95, 96, 100, 127, 128, 255, 256, 1023, 4096)

	for _, off := range offsets {
		for _, mlen := range mlens {
			prefix := max(off, 16)
			src, want := buildSingleMatchBlock(prefix, off, mlen)
			dst := make([]byte, len(want))
			n := decodeBlock(dst, src, nil)
			if n != len(want) {
				t.Fatalf("off=%d mlen=%d: decode returned %d, want %d",
					off, mlen, n, len(want))
			}
			if !bytes.Equal(dst[:n], want) {
				for i := range n {
					if dst[i] != want[i] {
						t.Fatalf("off=%d mlen=%d: first mismatch at byte %d: got 0x%02x want 0x%02x",
							off, mlen, i, dst[i], want[i])
					}
				}
			}
		}
	}
}

// TestMatchCopyChained runs many matches back-to-back so a bug that leaks
// register state between match-copy invocations shows up as corruption in a
// later match — which is exactly the kind of bug
// TestCompressUncompressBlock only finds incidentally.
func TestMatchCopyChained(t *testing.T) {
	const prefix = 256
	var buf bytes.Buffer

	want := make([]byte, prefix)
	for i := range want {
		want[i] = byte(i%253 + 1)
	}

	// Seq 0: entire prefix as literals + a first large match.
	{
		off := 64
		mlen := 200
		writeToken(&buf, prefix, mlen-minMatch)
		buf.Write(want[:prefix])
		off2 := make([]byte, 2)
		binary.LittleEndian.PutUint16(off2, uint16(off))
		buf.Write(off2)
		if rawM := mlen - minMatch; rawM >= 15 {
			writeExtended(&buf, rawM-15)
		}
		start := len(want)
		for i := range mlen {
			want = append(want, want[start-off+i])
		}
	}

	// Many follow-on matches with no literals between them, varying offset
	// and match length across thresholds.
	for i := range 40 {
		off := 8 + (i*3)%120 // sweep small + medium + large offsets
		mlen := 16 + (i*7)%200
		writeToken(&buf, 0, mlen-minMatch)
		off2 := make([]byte, 2)
		binary.LittleEndian.PutUint16(off2, uint16(off))
		buf.Write(off2)
		if rawM := mlen - minMatch; rawM >= 15 {
			writeExtended(&buf, rawM-15)
		}
		start := len(want)
		for j := range mlen {
			want = append(want, want[start-off+j])
		}
	}

	// Trailing literal-only sequence so the block ends correctly.
	tail := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	writeToken(&buf, len(tail), 0)
	buf.Write(tail)
	want = append(want, tail...)

	dst := make([]byte, len(want))
	n := decodeBlock(dst, buf.Bytes(), nil)
	if n != len(want) {
		t.Fatalf("decode returned %d, want %d", n, len(want))
	}
	if !bytes.Equal(dst[:n], want) {
		for i := range n {
			if dst[i] != want[i] {
				t.Fatalf("first mismatch at byte %d: got 0x%02x want 0x%02x", i, dst[i], want[i])
			}
		}
	}
}
