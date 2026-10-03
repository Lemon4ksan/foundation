// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fskit_test

import (
	"fmt"
	"io/fs"

	"github.com/lemon4ksan/foundation/fskit"
)

func ExampleFastWalk() {
	err := fskit.FastWalk(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			// Found file
			_ = path
		}
		return nil
	})
	_ = err
}

func ExampleOpenMmap() {
	// Memory map large dataset
	mapping, err := fskit.OpenMmap("large_dataset.bin")
	if err != nil {
		return
	}
	defer mapping.Close()

	data := mapping.Bytes()
	fmt.Printf("Mapped %d bytes\n", len(data))
}
