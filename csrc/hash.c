// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include <immintrin.h>
#include <stdint.h>
#include <stddef.h>

// hash64_avx2 processes 32-byte chunks using 256-bit AVX2 vector operations.
uint64_t hash64_avx2(const uint8_t* data, uint64_t len, uint64_t seed) {
    size_t i = 0;
    uint64_t h = seed ^ (len * 0x9E3779B185EBCA87ULL);

    for (; i + 32 <= len; i += 32) {
        __m256i chunk = _mm256_loadu_si256((const __m256i*)(data + i));
        
        // Extract 4 x 64-bit lanes
        uint64_t v0 = (uint64_t)_mm256_extract_epi64(chunk, 0);
        uint64_t v1 = (uint64_t)_mm256_extract_epi64(chunk, 1);
        uint64_t v2 = (uint64_t)_mm256_extract_epi64(chunk, 2);
        uint64_t v3 = (uint64_t)_mm256_extract_epi64(chunk, 3);

        h ^= v0 * 0xC2B2AE3D27D4EB4FULL;
        h = (h << 27) | (h >> 37);
        h += v1 * 0x85EBCA77C2B2AE63ULL;
        h = (h << 31) | (h >> 33);
        h ^= v2 * 0x165667B19E3779F9ULL;
        h = (h << 23) | (h >> 41);
        h += v3 * 0x27D4EB2F165667C5ULL;
        h = (h << 29) | (h >> 35);
    }

    for (; i + 8 <= len; i += 8) {
        uint64_t v = *(const uint64_t*)(data + i);
        h ^= v * 0xC2B2AE3D27D4EB4FULL;
        h = (h << 27) | (h >> 37);
    }

    for (; i < len; i++) {
        h ^= (uint64_t)data[i] * 0x27D4EB2F165667C5ULL;
        h = (h << 11) | (h >> 53);
    }

    h ^= h >> 33;
    h *= 0xC2B2AE3D27D4EB4FULL;
    h ^= h >> 29;
    h *= 0x85EBCA77C2B2AE63ULL;
    h ^= h >> 32;

    return h;
}
