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

// to_lower_ascii_avx2 converts ASCII characters in src to lowercase in dst.
void to_lower_ascii_avx2(const uint8_t* src, uint64_t len, uint8_t* dst) {
    size_t i = 0;
    __m256i upper_a = load_imm32_ymm(0x41414141); // 'A'
    __m256i upper_z = load_imm32_ymm(0x5A5A5A5A); // 'Z'
    __m256i delta   = load_imm32_ymm(0x20202020); // 32 ('a' - 'A')

    for (; i + 32 <= len; i += 32) {
        __m256i chunk = _mm256_loadu_si256((const __m256i*)(src + i));

        // Mask for bytes >= 'A' and <= 'Z'
        __m256i ge_a = _mm256_cmpgt_epi8(chunk, _mm256_sub_epi8(upper_a, load_imm32_ymm(0x01010101)));
        __m256i le_z = _mm256_cmpgt_epi8(_mm256_add_epi8(upper_z, load_imm32_ymm(0x01010101)), chunk);
        __m256i mask = _mm256_and_si256(ge_a, le_z);

        __m256i lower = _mm256_add_epi8(chunk, _mm256_and_si256(mask, delta));
        _mm256_storeu_si256((__m256i*)(dst + i), lower);
    }

    for (; i < len; i++) {
        uint8_t c = src[i];
        if (c >= 'A' && c <= 'Z') {
            c += 32;
        }
        dst[i] = c;
    }
}

// to_upper_ascii_avx2 converts ASCII characters in src to uppercase in dst.
void to_upper_ascii_avx2(const uint8_t* src, uint64_t len, uint8_t* dst) {
    size_t i = 0;
    __m256i lower_a = load_imm32_ymm(0x61616161); // 'a'
    __m256i lower_z = load_imm32_ymm(0x7A7A7A7A); // 'z'
    __m256i delta   = load_imm32_ymm(0x20202020); // 32 ('a' - 'A')

    for (; i + 32 <= len; i += 32) {
        __m256i chunk = _mm256_loadu_si256((const __m256i*)(src + i));

        // Mask for bytes >= 'a' and <= 'z'
        __m256i ge_a = _mm256_cmpgt_epi8(chunk, _mm256_sub_epi8(lower_a, load_imm32_ymm(0x01010101)));
        __m256i le_z = _mm256_cmpgt_epi8(_mm256_add_epi8(lower_z, load_imm32_ymm(0x01010101)), chunk);
        __m256i mask = _mm256_and_si256(ge_a, le_z);

        __m256i upper = _mm256_sub_epi8(chunk, _mm256_and_si256(mask, delta));
        _mm256_storeu_si256((__m256i*)(dst + i), upper);
    }

    for (; i < len; i++) {
        uint8_t c = src[i];
        if (c >= 'a' && c <= 'z') {
            c -= 32;
        }
        dst[i] = c;
    }
}
