package sign_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/crypto/sign"
)

func ExampleGenerateKey() {
	// For deterministic output, use NewKeyFromSeed
	seed := make([]byte, sign.SeedSize)
	pub, priv, _ := sign.NewKeyFromSeed(seed)

	msg := []byte("signed data")
	signature := sign.Sign(priv, msg)
	valid := sign.Verify(pub, msg, signature)

	fmt.Printf("Signature valid: %v\n", valid)
	// Output: Signature valid: true
}
