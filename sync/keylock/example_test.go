package keylock_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/lemon4ksan/foundation/sync/keylock"
)

func ExampleKeyMutex() {
	kl := keylock.New[string]()
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		kl.Lock("user-alice")
		defer kl.Unlock("user-alice")
		fmt.Println("Locked alice")
		time.Sleep(50 * time.Millisecond)
	}()

	go func() {
		defer wg.Done()
		kl.Lock("user-bob")
		defer kl.Unlock("user-bob")
		fmt.Println("Locked bob")
		time.Sleep(50 * time.Millisecond)
	}()

	wg.Wait()
	// Unordered output:
	// Locked alice
	// Locked bob
}
