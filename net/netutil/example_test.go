package netutil_test

import (
	"fmt"
	"github.com/lemon4ksan/foundation/net/netutil"
)

func ExampleCleanHost() {
	// Strips IPv6 zone IDs and converts IDNs.
	fmt.Println(netutil.CleanHost("fe80::1%eth0"))
	fmt.Println(netutil.CleanHost("президент.рф"))

	// Output:
	// fe80::1
	// xn--d1abbgf6aiiy.xn--p1ai
}

func ExampleCleanHostPort() {
	host, port := netutil.CleanHostPort("[fe80::1%eth0]:8080")
	fmt.Println("Host:", host)
	fmt.Println("Port:", port)

	// Output:
	// Host: fe80::1
	// Port: 8080
}
