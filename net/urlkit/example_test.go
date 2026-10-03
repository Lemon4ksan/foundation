package urlkit_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/net/urlkit"
)

func ExampleParseView() {
	u, err := urlkit.ParseView("https://api.internal/v1/users?id=123")
	if err == nil {
		fmt.Println("Host:", u.Host)
		fmt.Println("Path:", u.Path)
	}

	// Output:
	// Host: api.internal
	// Path: /v1/users
}
