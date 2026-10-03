// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randkit_test

import (
	"fmt"
	"time"

	"github.com/lemon4ksan/foundation/silicon/randkit"
)

func ExampleAppendUUIDv7() {
	var uuidBuf []byte
	uuidStr := randkit.AppendUUIDv7(uuidBuf, time.Now())

	fmt.Printf("UUID length: %d\n", len(uuidStr))

	// Output:
	// UUID length: 36
}
