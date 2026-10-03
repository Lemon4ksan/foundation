// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diff_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/text/diff"
)

func ExampleDiffReport() {
	report := &diff.DiffReport{
		LocalTarget:           "local-api",
		RemoteTarget:          "remote-openapi",
		TotalEndpointsChecked: 1,
		Drifts: []diff.DriftItem{
			{
				Severity: diff.SeverityGhost,
				Kind:     diff.DriftMissingEndpoint,
				Endpoint: "GET /api/v1/users",
				Message:  "Endpoint missing in remote spec",
			},
		},
	}

	fmt.Println(report.HasBreaking())
	fmt.Println(report.GhostCount())
	// Output:
	// false
	// 1
}
