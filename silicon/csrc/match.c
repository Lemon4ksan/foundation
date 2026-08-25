// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include <immintrin.h>
#include <stdint.h>
#include <stddef.h>

// find_match_length_avx2 finds how many bytes at a and b match consecutively (up to max_len).
// Used as the core LZ77 sliding window match finder for Brotli and Deflate compression.
uint64_t find_match_length_avx2(const uint8_t* a, const uint8_t* b, uint64_t max_len) {
    size_t match = 0;

    for (; match + 32 <= max_len; match += 32) {
        __m256i va = _mm256_loadu_si256((const __m256i*)(a + match));
        __m256i vb = _mm256_loadu_si256((const __m256i*)(b + match));

        __m256i cmp = _mm256_cmpeq_epi8(va, vb);
        uint32_t mask = (uint32_t)_mm256_movemask_epi8(cmp);

        if (mask != 0xFFFFFFFF) {
            uint32_t mismatch = (uint32_t)__builtin_ctz(~mask);
            return match + mismatch;
        }
    }

    for (; match < max_len; match++) {
        if (a[match] != b[match]) {
            break;
        }
    }

    return match;
}
