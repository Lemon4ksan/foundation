package ipc_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/net/ipc"
)

func ExampleParseIPCURI() {
	sock, req, isIPC := ipc.ParseIPCURI("unix:///var/run/docker.sock/v1.41/containers/json")
	if isIPC {
		fmt.Printf("Socket: %s\n", sock)
		fmt.Printf("RequestURI: %s\n", req)
	}

	// Output:
	// Socket: /var/run/docker.sock
	// RequestURI: /v1.41/containers/json
}
