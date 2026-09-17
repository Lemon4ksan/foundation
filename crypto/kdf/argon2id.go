// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kdf

import (
	"golang.org/x/crypto/argon2"
)

// Argon2idProfile defines memory hardness, iteration count, and parallelism for Argon2id.
type Argon2idProfile struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
}

var (
	// ProfileFast is optimized for fast unit tests and low-latency environments.
	ProfileFast = Argon2idProfile{MemoryKiB: 1024, Iterations: 1, Parallelism: 1}

	// ProfileInteractive provides OWASP-recommended interactive login security (~64MB RAM).
	ProfileInteractive = Argon2idProfile{MemoryKiB: 64 * 1024, Iterations: 3, Parallelism: 4}

	// ProfileModerate provides elevated security for sensitive local vaults (~128MB RAM).
	ProfileModerate = Argon2idProfile{MemoryKiB: 128 * 1024, Iterations: 4, Parallelism: 4}

	// ProfileSensitive provides high-security encryption against GPU/ASIC attacks (~256MB RAM).
	ProfileSensitive = Argon2idProfile{MemoryKiB: 256 * 1024, Iterations: 5, Parallelism: 8}

	// ProfileParanoid provides maximum memory hardness for long-term cold archives (~512MB RAM).
	ProfileParanoid = Argon2idProfile{MemoryKiB: 512 * 1024, Iterations: 8, Parallelism: 8}
)

// Argon2id derives a cryptographic key from password and salt using the specified profile parameters.
func Argon2id(password, salt []byte, profile Argon2idProfile, keyLen uint32) []byte {
	return argon2.IDKey(password, salt, profile.Iterations, profile.MemoryKiB, profile.Parallelism, keyLen)
}
