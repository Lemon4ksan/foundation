package compress_test

import (
	"fmt"
	"log"

	"github.com/lemon4ksan/foundation/codec/compress"
)

func ExampleDecompress() {
	// A sample payload to compress and then decompress
	payload := []byte("hello world")

	// Compress the payload using gzip
	compressedData, err := compress.CompressGzip(payload, nil)
	if err != nil {
		log.Fatalf("Compression error: %v", err)
	}

	// Decompress payload transparently using the specified algorithm
	decompressed, err := compress.Decompress("gzip", compressedData, nil)
	if err != nil {
		log.Fatalf("Decompression error: %v", err)
	}

	fmt.Printf("Decompressed: %s\n", decompressed)
	// Output: Decompressed: hello world
}
