package pipeline_test

import (
	"context"
	"fmt"

	"github.com/lemon4ksan/foundation/async/pipeline"
)

type (
	UserID      string
	UserProfile struct {
		ID   UserID
		Tier string
	}
)

func ExampleMap() {
	ctx := context.Background()
	userIDs := []UserID{"u1", "u2", "u3", "u4", "u5"}

	profiles, err := pipeline.Map(ctx, pipeline.Config{
		Workers:  5,
		RPS:      100.0, // 100 req/sec limit
		Burst:    10,
		FailFast: true,
	}, userIDs, func(ctx context.Context, id UserID) (UserProfile, error) {
		return UserProfile{ID: id, Tier: "premium"}, nil
	})
	if err != nil {
		panic(err)
	}

	for _, p := range profiles {
		fmt.Printf("User %s is %s\n", p.ID, p.Tier)
	}
	// Output:
	// User u1 is premium
	// User u2 is premium
	// User u3 is premium
	// User u4 is premium
	// User u5 is premium
}
