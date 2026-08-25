// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include <immintrin.h>
#include <stdint.h>
#include <stddef.h>

static inline __m256i load_imm32_ymm(uint32_t imm) {
    __m128i out;
    __asm__ (
        "vmovd %1, %0\n\t"
        "vpbroadcastd %0, %0"
        : "=x"(out)
        : "r"(imm)
    );
    return _mm256_broadcastsi128_si256(out);
}

static inline int from_hex(uint8_t c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    return -1;
}

// url_unescape_avx2 unescapes URL percent-encoded data (%XX) and '+' -> ' ' into dst.
// Returns the number of unescaped bytes, or -1 on invalid %XX escape sequence.
int64_t url_unescape_avx2(const uint8_t* src, uint64_t len, uint8_t* dst) {
    size_t i = 0;
    size_t out = 0;

    __m256i percent_const = load_imm32_ymm(0x25252525); // '%'
    __m256i plus_const    = load_imm32_ymm(0x2B2B2B2B); // '+'

    for (; i + 32 <= len; ) {
        __m256i chunk = _mm256_loadu_si256((const __m256i*)(src + i));

        __m256i is_percent = _mm256_cmpeq_epi8(chunk, percent_const);
        __m256i is_plus    = _mm256_cmpeq_epi8(chunk, plus_const);

        __m256i is_special = _mm256_or_si256(is_percent, is_plus);
        uint32_t mask = (uint32_t)_mm256_movemask_epi8(is_special);

        if (mask == 0) {
            // Fast-path: 32 clean bytes, copy directly
            _mm256_storeu_si256((__m256i*)(dst + out), chunk);
            i += 32;
            out += 32;
            continue;
        }

        // Process bytes until the end of this chunk
        size_t end = i + 32;
        while (i < end) {
            uint8_t c = src[i];
            if (c == '%') {
                if (i + 2 >= len) return -1;
                int hi = from_hex(src[i + 1]);
                int lo = from_hex(src[i + 2]);
                if (hi < 0 || lo < 0) return -1;
                dst[out++] = (uint8_t)((hi << 4) | lo);
                i += 3;
            } else if (c == '+') {
                dst[out++] = ' ';
                i++;
            } else {
                dst[out++] = c;
                i++;
            }
        }
    }

    // Remainder loop
    while (i < len) {
        uint8_t c = src[i];
        if (c == '%') {
            if (i + 2 >= len) return -1;
            int hi = from_hex(src[i + 1]);
            int lo = from_hex(src[i + 2]);
            if (hi < 0 || lo < 0) return -1;
            dst[out++] = (uint8_t)((hi << 4) | lo);
            i += 3;
        } else if (c == '+') {
            dst[out++] = ' ';
            i++;
        } else {
            dst[out++] = c;
            i++;
        }
    }

    return (int64_t)out;
}
