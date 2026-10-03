package varint_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/encoding/varint"
)

func ExampleParse() {
	// Example QUIC varint bytes (representing 151288809941952652)
	data := []byte{0xc2, 0x19, 0x7c, 0x5e, 0xff, 0x14, 0xe8, 0x8c}

	val, bytesRead, err := varint.Parse(data)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("Value: %d, Bytes Read: %d\n", val, bytesRead)
	// Output: Value: 151288809941952652, Bytes Read: 8
}

func ExampleAppend() {
	var buf []byte
	buf = varint.Append(buf, 15293)

	fmt.Printf("Appended bytes length: %d\n", len(buf))
	// Output: Appended bytes length: 2
}
