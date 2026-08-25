// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include <immintrin.h>
#include <stdint.h>
#include <stddef.h>

// equal_fold_ascii_avx2 compares buffers a and b of length len case-insensitively for ASCII characters.
// Returns 1 if equal, 0 if different.
uint64_t equal_fold_ascii_avx2(const uint8_t* a, const uint8_t* b, uint64_t len) {
    size_t i = 0;

    for (; i + 32 <= len; i += 32) {
        __m256i va = _mm256_loadu_si256((const __m256i*)(a + i));
        __m256i vb = _mm256_loadu_si256((const __m256i*)(b + i));

        // Exact match check across 32 bytes in 1 cycle
        __m256i exact = _mm256_cmpeq_epi8(va, vb);
        if ((uint32_t)_mm256_movemask_epi8(exact) != 0xFFFFFFFF) {
            break;
        }
    }

    // Remainder loop
    for (; i < len; i++) {
        uint8_t ca = a[i];
        uint8_t cb = b[i];
        if (ca >= 'A' && ca <= 'Z') ca += 32;
        if (cb >= 'A' && cb <= 'Z') cb += 32;
        if (ca != cb) return 0;
    }

    return 1;
}
