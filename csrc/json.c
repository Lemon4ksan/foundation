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

// json_skip_whitespace_avx2 advances cursor past whitespace characters (' ', '\t', '\r', '\n').
uint64_t json_skip_whitespace_avx2(const uint8_t* data, uint64_t len, uint64_t cursor) {
    size_t i = cursor;

    __m256i is_space_const = load_imm32_ymm(0x20202020); // ' '
    __m256i is_tab_const   = load_imm32_ymm(0x09090909); // '\t'
    __m256i is_cr_const    = load_imm32_ymm(0x0D0D0D0D); // '\r'
    __m256i is_lf_const    = load_imm32_ymm(0x0A0A0A0A); // '\n'

    for (; i + 32 <= len; i += 32) {
        __m256i chunk = _mm256_loadu_si256((const __m256i*)(data + i));

        __m256i is_space = _mm256_cmpeq_epi8(chunk, is_space_const);
        __m256i is_tab   = _mm256_cmpeq_epi8(chunk, is_tab_const);
        __m256i is_cr    = _mm256_cmpeq_epi8(chunk, is_cr_const);
        __m256i is_lf    = _mm256_cmpeq_epi8(chunk, is_lf_const);

        __m256i is_ws = _mm256_or_si256(_mm256_or_si256(is_space, is_tab), _mm256_or_si256(is_cr, is_lf));
        uint32_t mask = (uint32_t)_mm256_movemask_epi8(is_ws);

        if (mask != 0xFFFFFFFF) {
            uint32_t non_ws_offset = (uint32_t)__builtin_ctz(~mask);
            return i + non_ws_offset;
        }
    }

    for (; i < len; i++) {
        uint8_t b = data[i];
        if (b != ' ' && b != '\t' && b != '\r' && b != '\n') {
            return i;
        }
    }

    return len;
}

// json_scan_string_avx2 scans for the first occurrence of '"' or '\' in data starting at cursor.
// Returns the index of the special character, or -1 if not found.
int64_t json_scan_string_avx2(const uint8_t* data, uint64_t len, uint64_t cursor) {
    size_t i = cursor;

    __m256i is_quote_const = load_imm32_ymm(0x22222222); // '"'
    __m256i is_slash_const = load_imm32_ymm(0x5C5C5C5C); // '\\'

    for (; i + 32 <= len; i += 32) {
        __m256i chunk = _mm256_loadu_si256((const __m256i*)(data + i));

        __m256i is_quote = _mm256_cmpeq_epi8(chunk, is_quote_const);
        __m256i is_slash = _mm256_cmpeq_epi8(chunk, is_slash_const);

        __m256i is_special = _mm256_or_si256(is_quote, is_slash);
        uint32_t mask = (uint32_t)_mm256_movemask_epi8(is_special);

        if (mask != 0) {
            uint32_t match_offset = (uint32_t)__builtin_ctz(mask);
            return (int64_t)(i + match_offset);
        }
    }

    for (; i < len; i++) {
        uint8_t b = data[i];
        if (b == '"' || b == '\\') {
            return (int64_t)i;
        }
    }

    return -1;
}
