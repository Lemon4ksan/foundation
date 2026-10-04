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
			_ = path
		}
		return nil
	})
	_ = err
}

func ExampleFastWalkSeq() {
	for path, entry := range fskit.FastWalkSeq(".") {
		if !entry.IsDir() {
			_ = path
		}
	}
}

func ExampleOpenMmap() {
	mapping, err := fskit.OpenMmap("large_dataset.bin")
	if err != nil {
		return
	}
	defer mapping.Close()

	data := mapping.Bytes()
	fmt.Printf("Mapped %d bytes\n", len(data))
}

func ExampleIsIncompressibleExtension() {
	isCompressed := fskit.IsIncompressibleExtension("archive.tar.gz")
	fmt.Println(isCompressed)
	// Output:
	// true
}

func ExampleAtomicWriteFile() {
	err := fskit.AtomicWriteFile("config.json", []byte(`{"version": 1}`), 0o644)
	_ = err
}
