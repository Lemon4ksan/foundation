package bin_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/encoding/bin"
)

func ExampleReader() {
	payload := []byte{0xAB, 0xCD, 0x01, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x04, 0xFF, 0x01, 0x02, 0x03, 0x04}
	r := bin.NewReader(payload)

	magic := r.U16BE()
	_ = r.U32BE() // StreamID
	length := r.U32BE()
	_ = r.U8() // Flags
	body := r.Bytes(int(length))

	if err := r.Err(); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("Magic: %X, Length: %d, Data len: %d\n", magic, length, len(body))
	// Output: Magic: ABCD, Length: 4, Data len: 4
}
