// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event_test

import (
	"fmt"
	"time"

	"github.com/lemon4ksan/foundation/async/event"
)

type UserRegistered struct {
	event.Envelope
	UserID string
	Email  string
}

func ExampleBus_Subscribe() {
	bus := event.New()
	defer bus.Close()

	// Type-safe subscription
	sub := bus.Subscribe(UserRegistered{})
	defer sub.Unsubscribe()

	// Consumer loop
	go func() {
		for ev := range sub.C() {
			userEv := ev.(UserRegistered)
			fmt.Printf("Received: %s\n", userEv.UserID)
		}
	}()

	// Publisher emits event without blocking
	bus.Publish(UserRegistered{
		UserID: "usr_5501",
		Email:  "alice@example.com",
	})

	time.Sleep(20 * time.Millisecond)

	// Output:
	// Received: usr_5501
}
