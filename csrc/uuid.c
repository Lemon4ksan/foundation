// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include <stdint.h>
#include <stddef.h>

static inline uint8_t byte_to_hex(uint8_t nibble) {
    return nibble < 10 ? nibble + '0' : nibble - 10 + 'a';
}

static inline int from_hex(uint8_t c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    return -1;
}

// uuid_format_avx2 formats a 16-byte binary UUID into standard 36-byte hex-and-dash format.
void uuid_format_avx2(const uint8_t* src16, uint8_t* dst36) {
    size_t out_pos = 0;
    for (size_t i = 0; i < 16; i++) {
        if (out_pos == 8 || out_pos == 13 || out_pos == 18 || out_pos == 23) {
            dst36[out_pos++] = '-';
        }
        uint8_t b = src16[i];
        dst36[out_pos++] = byte_to_hex(b >> 4);
        dst36[out_pos++] = byte_to_hex(b & 0x0F);
    }
}

// uuid_parse_avx2 parses and validates standard 36-byte UUID string into 16-byte output.
// Returns 1 if valid, 0 if invalid format.
uint64_t uuid_parse_avx2(const uint8_t* src36, uint8_t* dst16) {
    if (src36[8] != '-' || src36[13] != '-' || src36[18] != '-' || src36[23] != '-') {
        return 0;
    }

    size_t in_pos = 0;
    for (size_t i = 0; i < 16; i++) {
        if (in_pos == 8 || in_pos == 13 || in_pos == 18 || in_pos == 23) {
            in_pos++;
        }
        int hi = from_hex(src36[in_pos]);
        int lo = from_hex(src36[in_pos + 1]);
        if (hi < 0 || lo < 0) return 0;
        dst16[i] = (uint8_t)((hi << 4) | lo);
        in_pos += 2;
    }

    return 1;
}
