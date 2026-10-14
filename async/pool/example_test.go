// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool_test

import (
	"context"
	"fmt"
	"time"

	"github.com/lemon4ksan/foundation/async/pool"
)

type ProcessedImage struct {
	Width  int
	Height int
	Data   []byte
}

func ExamplePool() {
	ctx := context.Background()

	imagePool := pool.New[ProcessedImage](pool.Config{
		MinWorkers:  2,
		MaxWorkers:  16,
		IdleTimeout: 5 * time.Second,
		QueueLimit:  200,
	})
	defer imagePool.Close()

	future, err := imagePool.Submit(ctx, func(taskCtx context.Context) (ProcessedImage, error) {
		time.Sleep(20 * time.Millisecond) // Image resizing
		return ProcessedImage{Width: 1920, Height: 1080, Data: []byte("resized")}, nil
	})
	if err != nil {
		panic(err)
	}

	img, err := future.Get(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Image processed: %dx%d\n", img.Width, img.Height)
	// Output:
	// Image processed: 1920x1080
}
