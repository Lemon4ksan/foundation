// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package breaker_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lemon4ksan/foundation/sync/breaker"
)

type UserProfile struct {
	ID   string
	Name string
}

func fetchRemoteProfile(ctx context.Context) (UserProfile, error) {
	return UserProfile{ID: "1", Name: "Remote User"}, nil
}

func getCachedProfile() UserProfile {
	return UserProfile{ID: "1", Name: "Cached User"}
}

func ExampleCircuitBreaker() {
	cb := breaker.New[UserProfile](breaker.Config{
		FailureThreshold: 0.5,              // Trip if 50% of calls fail
		Cooldown:         10 * time.Second, // Test Half-Open after 10s
		MinRequests:      5,                // Minimum sample size
	})

	ctx := context.Background()
	profile, err := cb.Do(ctx, func(ctx context.Context) (UserProfile, error) {
		return fetchRemoteProfile(ctx)
	})
	if err != nil {
		if errors.Is(err, breaker.ErrCircuitOpen) {
			// Fast-path fallback
			profile = getCachedProfile()
			err = nil
		}
	}

	fmt.Println(profile.Name, err)
	// Output:
	// Remote User <nil>
}
