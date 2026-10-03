package secret_test

import (
	"fmt"
	"log/slog"

	"github.com/lemon4ksan/foundation/types/secret"
)

func ExampleSecret() {
	apiKey := secret.New("sk_live_983274982374982374")

	// Formatted printing masks the value
	fmt.Println("Key:", apiKey)

	// Structured logging masks the value
	slog.Info("Authenticating client", "api_key", apiKey)

	// Explicit extraction for authorized network requests
	rawKey := apiKey.Value()
	_ = rawKey
	// Output:
	// Key: ******
}
