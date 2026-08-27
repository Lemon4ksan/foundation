// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpukit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/cpukit"
	"github.com/lemon4ksan/foundation/testkit/assert"
)

func TestCPUKitProbes(t *testing.T) {
	t.Parallel()

	_ = cpukit.HasAVX2()
	_ = cpukit.HasBMI1()
	_ = cpukit.HasBMI2()
	_ = cpukit.HasBMI()
	_ = cpukit.HasAVX512()
	_ = cpukit.HasNEON()
	_ = cpukit.HasAESNI()

	cls := cpukit.CacheLineSize()
	assert.GreaterOrEqual(t, cls, 32)
}
