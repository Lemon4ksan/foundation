// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enctest

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/text/encoding"
	"github.com/lemon4ksan/foundation/text/encoding/charmap"
	"github.com/lemon4ksan/foundation/text/encoding/unicode"
)

func TestNopEncoding(t *testing.T) {
	TestEncoding(t, encoding.Nop, "hello world", "hello world", "", "")
}

func TestEncodingHarness(t *testing.T) {
	t.Run("Nop", func(t *testing.T) {
		TestEncoding(t, encoding.Nop, "the quick brown fox", "the quick brown fox", "", "")
	})

	t.Run("NopEmptyInput", func(t *testing.T) {
		TestEncoding(t, encoding.Nop, "", "", "", "")
	})

	t.Run("CharmapISO8859_1", func(t *testing.T) {
		TestEncoding(t, charmap.ISO8859_1, "\xe9\xe0\xe7", "éàç", "", "")
	})

	t.Run("UnicodeUTF8BOM", func(t *testing.T) {
		TestEncoding(t, unicode.UTF8BOM, "hello", "hello", "\xef\xbb\xbf", "")
	})
}

func TestFileHarness(t *testing.T) {
	t.Run("Nop", func(t *testing.T) {
		TestFile(t, encoding.Nop)
	})

	t.Run("ISO8859_1", func(t *testing.T) {
		TestFile(t, charmap.ISO8859_1)
	})

	t.Run("Windows1252", func(t *testing.T) {
		TestFile(t, charmap.Windows1252)
	})
}

func TestBenchmarkHarness(t *testing.T) {
	res := testing.Benchmark(func(b *testing.B) {
		Benchmark(b, encoding.Nop)
	})
	if res.N <= 0 {
		t.Fatalf("Benchmark(Nop) failed to run: N=%d", res.N)
	}

	resCharmap := testing.Benchmark(func(b *testing.B) {
		Benchmark(b, charmap.ISO8859_1)
	})
	if resCharmap.N <= 0 {
		t.Fatalf("Benchmark(ISO8859_1) failed to run: N=%d", resCharmap.N)
	}
}

func BenchmarkNop(b *testing.B) {
	Benchmark(b, encoding.Nop)
}

func BenchmarkISO8859_1(b *testing.B) {
	Benchmark(b, charmap.ISO8859_1)
}

func TestTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSame bool
		wantLen  int
	}{
		{
			name:     "Empty",
			input:    "",
			wantSame: true,
			wantLen:  0,
		},
		{
			name:     "Short",
			input:    "Hello, world!",
			wantSame: true,
			wantLen:  13,
		},
		{
			name:     "Under threshold 119",
			input:    strings.Repeat("a", 119),
			wantSame: true,
			wantLen:  119,
		},
		{
			name:     "At threshold 120",
			input:    strings.Repeat("b", 120),
			wantSame: false,
			wantLen:  103, // 50 + 3 + 50
		},
		{
			name:     "Long string 200",
			input:    strings.Repeat("x", 50) + strings.Repeat("y", 100) + strings.Repeat("z", 50),
			wantSame: false,
			wantLen:  103,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := trim(tc.input)
			if tc.wantSame {
				if got != tc.input {
					t.Fatalf("trim(%q) = %q, want exact input", tc.input, got)
				}
			} else {
				if len(got) != tc.wantLen {
					t.Fatalf("len(trim) = %d, want %d", len(got), tc.wantLen)
				}
				if !strings.HasPrefix(got, tc.input[:50]) {
					t.Fatalf("trim prefix mismatch: got %q", got[:50])
				}
				if !strings.HasSuffix(got, tc.input[len(tc.input)-50:]) {
					t.Fatalf("trim suffix mismatch: got %q", got[len(got)-50:])
				}
				if !strings.Contains(got, "...") {
					t.Fatalf("trim missing ellipsis: got %q", got)
				}
			}
		})
	}
}

func TestTranscoderInterface(t *testing.T) {
	var dec Transcoder = encoding.Nop.NewDecoder()
	var enc Transcoder = encoding.Nop.NewEncoder()

	if dec == nil || enc == nil {
		t.Fatalf("decoder or encoder does not satisfy Transcoder interface")
	}

	b, err := dec.Bytes([]byte("test"))
	if err != nil || string(b) != "test" {
		t.Fatalf("dec.Bytes failed: %v", err)
	}

	s, err := enc.String("test")
	if err != nil || s != "test" {
		t.Fatalf("enc.String failed: %v", err)
	}
}

type recordTester struct {
	failed bool
	msg    string
}

func (r *recordTester) Fatal(args ...any) {
	r.failed = true
	r.msg = fmt.Sprint(args...)
	runtime.Goexit()
}

func (r *recordTester) Fatalf(format string, args ...any) {
	r.failed = true
	r.msg = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

func runTestFn(fn func(t tester)) (failed bool, msg string) {
	var rt recordTester
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(&rt)
	}()
	<-done
	return rt.failed, rt.msg
}

type mockTransform struct {
	fn func(dst, src []byte, atEOF bool) (nDst, nSrc int, err error)
}

func (m *mockTransform) Reset() {}

func (m *mockTransform) Transform(dst, src []byte, atEOF bool) (int, int, error) {
	if m.fn != nil {
		return m.fn(dst, src, atEOF)
	}
	n := copy(dst, src)
	return n, n, nil
}

type dummyEncoding struct {
	dec *encoding.Decoder
	enc *encoding.Encoder
}

func (d *dummyEncoding) NewDecoder() *encoding.Decoder { return d.dec }
func (d *dummyEncoding) NewEncoder() *encoding.Encoder { return d.enc }

func TestHarnessErrorBranches(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("TransformError", func(t *testing.T) {
		enc := &dummyEncoding{
			dec: &encoding.Decoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					return 0, 0, errBoom
				},
			}},
		}
		failed, _ := runTestFn(func(rt tester) {
			testEncodingDirection(rt, enc, "Decode", "test", "test", "", "")
		})
		if !failed {
			t.Fatalf("expected failure on Transform error")
		}
	})

	t.Run("TransformDstCountMismatch", func(t *testing.T) {
		enc := &dummyEncoding{
			dec: &encoding.Decoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					return 0, len(src), nil
				},
			}},
		}
		failed, _ := runTestFn(func(rt tester) {
			testEncodingDirection(rt, enc, "Decode", "test", "test", "", "")
		})
		if !failed {
			t.Fatalf("expected failure on nDst mismatch")
		}
	})

	t.Run("TransformSrcCountMismatch", func(t *testing.T) {
		enc := &dummyEncoding{
			dec: &encoding.Decoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					copy(dst, src)
					return len(dst), 0, nil
				},
			}},
		}
		failed, _ := runTestFn(func(rt tester) {
			testEncodingDirection(rt, enc, "Decode", "test", "test", "", "")
		})
		if !failed {
			t.Fatalf("expected failure on nSrc mismatch")
		}
	})

	t.Run("TransformDstContentMismatch", func(t *testing.T) {
		enc := &dummyEncoding{
			dec: &encoding.Decoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					for i := range dst {
						dst[i] = 'X'
					}
					return len(dst), len(src), nil
				},
			}},
		}
		failed, _ := runTestFn(func(rt tester) {
			testEncodingDirection(rt, enc, "Decode", "test", "test", "", "")
		})
		if !failed {
			t.Fatalf("expected failure on content mismatch")
		}
	})

	t.Run("LoopStringError", func(t *testing.T) {
		calls := 0
		enc := &dummyEncoding{
			dec: &encoding.Decoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					calls++
					if calls > 1 {
						return 0, 0, errBoom
					}
					copy(dst, src)
					return len(dst), len(src), nil
				},
			}},
		}
		failed, _ := runTestFn(func(rt tester) {
			testEncodingDirection(rt, enc, "Decode", "test", "test", "", "")
		})
		if !failed {
			t.Fatalf("expected failure on loop String error")
		}
	})

	t.Run("LoopStringMismatch", func(t *testing.T) {
		calls := 0
		enc := &dummyEncoding{
			dec: &encoding.Decoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					calls++
					if calls > 1 {
						for i := range dst {
							dst[i] = 'Z'
						}
						return len(dst), len(src), nil
					}
					copy(dst, src)
					return len(dst), len(src), nil
				},
			}},
		}
		failed, _ := runTestFn(func(rt tester) {
			testEncodingDirection(rt, enc, "Decode", "test", "test", "", "")
		})
		if !failed {
			t.Fatalf("expected failure on loop String mismatch")
		}
	})

	t.Run("TestFileEncodeError", func(t *testing.T) {
		enc := &dummyEncoding{
			enc: &encoding.Encoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					return 0, 0, errBoom
				},
			}},
		}
		failed, _ := runTestFn(func(rt tester) {
			testFile(rt, enc)
		})
		if !failed {
			t.Fatalf("expected failure on TestFile encode error")
		}
	})

	t.Run("TestFileDecodeError", func(t *testing.T) {
		enc := &dummyEncoding{
			enc: &encoding.Encoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					copy(dst, src)
					return len(src), len(src), nil
				},
			}},
			dec: &encoding.Decoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					return 0, 0, errBoom
				},
			}},
		}
		failed, _ := runTestFn(func(rt tester) {
			testFile(rt, enc)
		})
		if !failed {
			t.Fatalf("expected failure on TestFile decode error")
		}
	})

	t.Run("TestFileMismatch", func(t *testing.T) {
		enc := &dummyEncoding{
			enc: &encoding.Encoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					copy(dst, src)
					return len(src), len(src), nil
				},
			}},
			dec: &encoding.Decoder{Transformer: &mockTransform{
				fn: func(dst, src []byte, atEOF bool) (int, int, error) {
					for i := range dst {
						dst[i] = 'Y'
					}
					return len(src), len(src), nil
				},
			}},
		}
		failed, _ := runTestFn(func(rt tester) {
			testFile(rt, enc)
		})
		if !failed {
			t.Fatalf("expected failure on TestFile mismatch")
		}
	})
}
