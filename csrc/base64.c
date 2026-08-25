// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include <stdint.h>
#include <stddef.h>

static inline uint8_t b64_char(uint8_t val) {
    if (val < 26) return (uint8_t)('A' + val);
    if (val < 52) return (uint8_t)('a' + (val - 26));
    if (val < 62) return (uint8_t)('0' + (val - 52));
    if (val == 62) return '+';
    return '/';
}

static inline int b64_val(uint8_t c) {
    if (c >= 'A' && c <= 'Z') return c - 'A';
    if (c >= 'a' && c <= 'z') return c - 'a' + 26;
    if (c >= '0' && c <= '9') return c - '0' + 52;
    if (c == '+' || c == '-') return 62;
    if (c == '/' || c == '_') return 63;
    if (c == '=') return 0;
    return -1;
}

// base64_encode_avx2 encodes src bytes into standard Base64 in dst.
// Returns the number of bytes written to dst.
uint64_t base64_encode_avx2(const uint8_t* src, uint64_t len, uint8_t* dst) {
    size_t i = 0;
    size_t out = 0;

    for (; i + 3 <= len; i += 3) {
        uint32_t triple = ((uint32_t)src[i] << 16) | ((uint32_t)src[i + 1] << 8) | (uint32_t)src[i + 2];
        dst[out++] = b64_char((triple >> 18) & 0x3F);
        dst[out++] = b64_char((triple >> 12) & 0x3F);
        dst[out++] = b64_char((triple >> 6) & 0x3F);
        dst[out++] = b64_char(triple & 0x3F);
    }

    size_t rem = len - i;
    if (rem == 1) {
        uint32_t triple = (uint32_t)src[i] << 16;
        dst[out++] = b64_char((triple >> 18) & 0x3F);
        dst[out++] = b64_char((triple >> 12) & 0x3F);
        dst[out++] = '=';
        dst[out++] = '=';
    } else if (rem == 2) {
        uint32_t triple = ((uint32_t)src[i] << 16) | ((uint32_t)src[i + 1] << 8);
        dst[out++] = b64_char((triple >> 18) & 0x3F);
        dst[out++] = b64_char((triple >> 12) & 0x3F);
        dst[out++] = b64_char((triple >> 6) & 0x3F);
        dst[out++] = '=';
    }

    return out;
}

// base64_decode_avx2 decodes Base64 data from src into dst.
// Returns the number of decoded bytes, or -1 on invalid format.
int64_t base64_decode_avx2(const uint8_t* src, uint64_t len, uint8_t* dst) {
    if (len == 0) return 0;
    if (len % 4 != 0) return -1;

    size_t i = 0;
    size_t out = 0;

    for (; i + 4 <= len; i += 4) {
        int v0 = b64_val(src[i]);
        int v1 = b64_val(src[i + 1]);
        int v2 = b64_val(src[i + 2]);
        int v3 = b64_val(src[i + 3]);

        if ((v0 | v1 | v2 | v3) < 0) return -1;

        uint32_t triple = ((uint32_t)v0 << 18) | ((uint32_t)v1 << 12) | ((uint32_t)v2 << 6) | (uint32_t)v3;

        dst[out++] = (uint8_t)(triple >> 16);
        if (src[i + 2] != '=') {
            dst[out++] = (uint8_t)(triple >> 8);
        }
        if (src[i + 3] != '=') {
            dst[out++] = (uint8_t)triple;
        }
    }

    return (int64_t)out;
}
