// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diff_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lemon4ksan/foundation/text/diff"
)

func TestDiffReport(t *testing.T) {
	t.Parallel()

	report := &diff.DiffReport{
		LocalTarget:           "local.go",
		RemoteTarget:          "remote.json",
		TotalEndpointsChecked: 10,
		Drifts: []diff.DriftItem{
			{
				Severity: diff.SeverityBreaking,
				Kind:     diff.DriftMissingParam,
				Endpoint: "POST /v1/users",
				Param:    "email",
				Message:  "missing required parameter email",
			},
			{
				Severity: diff.SeverityNonBreaking,
				Kind:     diff.DriftExtraParam,
				Endpoint: "GET /v1/users",
				Param:    "limit",
				Message:  "optional parameter limit added",
			},
			{
				Severity: diff.SeverityGhost,
				Kind:     diff.DriftMissingEndpoint,
				Endpoint: "DELETE /v1/legacy",
				Message:  "endpoint declared in local but missing in remote",
			},
		},
	}

	assert.Equal(t, 1, report.BreakingCount())
	assert.Equal(t, 1, report.NonBreakingCount())
	assert.Equal(t, 1, report.GhostCount())
	assert.True(t, report.HasBreaking())
	assert.Equal(t, 1, report.ExitCode())

	jsonBytes, err := report.JSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"severity": "BREAKING"`)
}
