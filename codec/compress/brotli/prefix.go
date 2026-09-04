package brotli

/* Copyright 2013 Google Inc. All Rights Reserved.

   Distributed under MIT license.
   See file LICENSE for detail or copy at https://opensource.org/licenses/MIT
*/

/* Functions for encoding of integers into prefix codes the amount of extra
   bits, and the actual values of the extra bits. */

/*
Here distance_code is an intermediate code, i.e. one of the special codes or

	the actual distance increased by BROTLI_NUM_DISTANCE_SHORT_CODES - 1.
*/
func prefixEncodeCopyDistance(distance_code, num_direct_codes, postfix_bits uint, code *uint16, extra_bits *uint32) {
	if distance_code < numDistanceShortCodes+num_direct_codes {
		*code = uint16(distance_code)
		*extra_bits = 0
		return
	} else {
		dist := (uint(1) << (postfix_bits + 2)) + (distance_code - numDistanceShortCodes - num_direct_codes)
		bucket := uint(log2FloorNonZero(dist) - 1)
		var postfix_mask uint = (1 << postfix_bits) - 1
		postfix := dist & postfix_mask
		prefix := (dist >> bucket) & 1
		offset := (2 + prefix) << bucket
		nbits := bucket - postfix_bits
		*code = uint16(
			nbits<<10 | (numDistanceShortCodes + num_direct_codes + ((2*(nbits-1) + prefix) << postfix_bits) + postfix),
		)
		*extra_bits = uint32((dist - offset) >> postfix_bits)
	}
}
