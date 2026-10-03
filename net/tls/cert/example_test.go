package cert_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/net/tls/cert"
)

func ExampleParseCompressionAlgorithm() {
	algo, err := cert.ParseCompressionAlgorithm("zstd")
	if err == nil {
		fmt.Println(algo)
	}

	// Output:
	// zstd
}
