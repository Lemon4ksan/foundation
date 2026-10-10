// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fskit

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AtomicWriteFile writes data to a temporary file in the directory of path
// and renames it atomically to path. It prevents concurrent readers or crashes
// from observing partial writes.
//
// It does not panic.
//
// Complexity: O(N) time where N is the length of data. Allocates memory for file operations.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".atomic-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// ReadFileLimited reads up to maxBytes from the specified file path.
// If the file size exceeds maxBytes, it returns an error before reading the full file.
//
// It does not panic.
//
// Complexity: O(N) time where N is the number of bytes read.
func ReadFileLimited(path string, maxBytes int64) ([]byte, error) {
	if maxBytes < 0 {
		return nil, errors.New("fskit: negative maxBytes limit")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() > maxBytes {
		return nil, fmt.Errorf("fskit: file size %d exceeds limit %d", fi.Size(), maxBytes)
	}

	lr := io.LimitReader(f, maxBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("fskit: file content exceeds limit %d", maxBytes)
	}
	return data, nil
}
