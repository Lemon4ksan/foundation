package lz4_test

import (
	"bytes"
	"fmt"
	"log"

	"github.com/lemon4ksan/foundation/codec/compress/lz4"
)

func ExampleCompressBlock() {
	data := []byte("compress this block with lz4 algorithm!")

	// Create a destination buffer large enough to hold the worst-case compressed size.
	maxSize := lz4.CompressBlockBound(len(data))
	dst := make([]byte, maxSize)

	compressedSize, err := lz4.CompressBlock(data, dst)
	if err != nil {
		log.Fatal(err)
	}
	compressed := dst[:compressedSize]

	// Decompress the block.
	uncompressed := make([]byte, len(data))
	uncompressedSize, err := lz4.UncompressBlock(compressed, uncompressed)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(bytes.Equal(data, uncompressed[:uncompressedSize]))
	// Output: true
}
