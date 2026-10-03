package aead_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/crypto/aead"
)

func ExampleNew() {
	key := make([]byte, aead.KeySize)
	// In reality, use crypto/rand for the key!

	c, err := aead.New(aead.ChaCha20Poly1305, key)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	nonce := make([]byte, 12)
	ciphertext, _ := aead.Seal(aead.ChaCha20Poly1305, key, nonce, []byte("secret"), nil)

	fmt.Printf("Encrypted length > 0: %v\n", len(ciphertext) > 0)
	fmt.Printf("Cipher ready: %v\n", c != nil)
	// Output:
	// Encrypted length > 0: true
	// Cipher ready: true
}
