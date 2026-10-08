// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit_test

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/lemon4ksan/foundation/tuikit"
)

func ExampleNewBox() {
	box := tuikit.NewBox("Status", 40)
	box.AddLine("Server online on :8080")
	box.AddLine("Healthy")
	box.Render(os.Stdout)
}

func ExampleTable() {
	table := tuikit.NewTable("Method", "Path", "Latency")
	table.AddRow("GET", "/api/v1/users", "1.2 ms")
	table.AddRow("POST", "/api/v1/login", "4.8 ms")
	table.Render(os.Stdout)
}

func ExampleFuncCommand() {
	cmd := &tuikit.FuncCommand{
		CmdName:     "ping",
		CmdSynopsis: "Send ping",
		CmdRun: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			fmt.Fprintln(stdout, "pong")
			return nil
		},
	}
	_ = cmd.Run(context.Background(), nil, os.Stdout, os.Stderr)
	// Output:
	// pong
}
