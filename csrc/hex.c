// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include <immintrin.h>
#include <stdint.h>
#include <stddef.h>

static inline __m128i load_imm32(uint32_t imm) {
    __m128i out;
    __asm__ (
        "vmovd %1, %0\n\t"
        "vpbroadcastd %0, %0"
        : "=x"(out)
        : "r"(imm)
    );
    return out;
}

// hex_encode_avx2 converts src bytes into lowercase hex ASCII in dst.
void hex_encode_avx2(const uint8_t* src, uint64_t len, uint8_t* dst) {
    if (len == 0) return;

    __m128i mask_lo    = load_imm32(0x0F0F0F0F);
    __m128i nine       = load_imm32(0x09090909);
    __m128i ascii_zero = load_imm32(0x30303030);
    __m128i diff       = load_imm32(0x27272727); // 'a' - '0' - 10 = 39 (0x27)

    size_t i = 0;
    size_t out_idx = 0;

    for (; i + 16 <= len; i += 16, out_idx += 32) {
        __m128i data = _mm_loadu_si128((const __m128i*)(src + i));

        __m128i hi_nibbles = _mm_and_si128(_mm_srli_epi16(data, 4), mask_lo);
        __m128i lo_nibbles = _mm_and_si128(data, mask_lo);

        // Convert hi nibbles
        __m128i hi_gt9 = _mm_cmpgt_epi8(hi_nibbles, nine);
        __m128i hi_base = _mm_add_epi8(hi_nibbles, ascii_zero);
        __m128i hi_ascii = _mm_add_epi8(hi_base, _mm_and_si128(hi_gt9, diff));

        // Convert lo nibbles
        __m128i lo_gt9 = _mm_cmpgt_epi8(lo_nibbles, nine);
        __m128i lo_base = _mm_add_epi8(lo_nibbles, ascii_zero);
        __m128i lo_ascii = _mm_add_epi8(lo_base, _mm_and_si128(lo_gt9, diff));

        __m128i hex_part1 = _mm_unpacklo_epi8(hi_ascii, lo_ascii);
        __m128i hex_part2 = _mm_unpackhi_epi8(hi_ascii, lo_ascii);

        _mm_storeu_si128((__m128i*)(dst + out_idx), hex_part1);
        _mm_storeu_si128((__m128i*)(dst + out_idx + 16), hex_part2);
    }

    for (; i < len; i++, out_idx += 2) {
        uint8_t b = src[i];
        uint8_t hi = b >> 4;
        uint8_t lo = b & 0x0F;
        dst[out_idx] = hi < 10 ? hi + '0' : hi - 10 + 'a';
        dst[out_idx + 1] = lo < 10 ? lo + '0' : lo - 10 + 'a';
    }
}

// hex_decode_avx2 decodes hex ASCII bytes from src into dst.
uint64_t hex_decode_avx2(const uint8_t* src, uint64_t len, uint8_t* dst) {
    if (len & 1) return 0;
    size_t i = 0;
    size_t out_idx = 0;

    for (; i < len; i += 2, out_idx++) {
        uint8_t c1 = src[i];
        uint8_t c2 = src[i + 1];
        uint8_t n1, n2;

        if (c1 >= '0' && c1 <= '9') n1 = c1 - '0';
        else if (c1 >= 'a' && c1 <= 'f') n1 = c1 - 'a' + 10;
        else if (c1 >= 'A' && c1 <= 'F') n1 = c1 - 'A' + 10;
        else return 0;

        if (c2 >= '0' && c2 <= '9') n2 = c2 - '0';
        else if (c2 >= 'a' && c2 <= 'f') n2 = c2 - 'a' + 10;
        else if (c2 >= 'A' && c2 <= 'F') n2 = c2 - 'A' + 10;
        else return 0;

        dst[out_idx] = (n1 << 4) | n2;
    }

    return 1;
}
