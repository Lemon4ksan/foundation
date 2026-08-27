// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diff_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
	"github.com/lemon4ksan/foundation/text/diff"
)

func TestDiffReport_Complete(t *testing.T) {
	t.Parallel()

	// 1. Clean report non-additive
	cleanReport := &diff.DiffReport{
		LocalTarget:           "local.go",
		RemoteTarget:          "remote.json",
		TotalEndpointsChecked: 5,
		Additive:              false,
		Drifts:                nil,
	}
	assert.Equal(t, 0, cleanReport.BreakingCount())
	assert.Equal(t, 0, cleanReport.NonBreakingCount())
	assert.Equal(t, 0, cleanReport.GhostCount())
	assert.False(t, cleanReport.HasBreaking())
	assert.False(t, cleanReport.HasDrift())
	assert.Equal(t, 0, cleanReport.ExitCode())
	assert.Contains(t, cleanReport.Render(false), "All contracts are 100% in sync")

	// 2. Clean report additive
	cleanAdditive := &diff.DiffReport{
		LocalTarget:           "local.go",
		RemoteTarget:          "remote.json",
		TotalEndpointsChecked: 5,
		Additive:              true,
		Drifts:                []diff.DriftItem{},
	}
	assert.Contains(t, cleanAdditive.Render(true), "100% satisfied by local contract")

	// 3. Report with breaking, non-breaking, ghost drifts, suggestions, and expected/actual
	fullReport := &diff.DiffReport{
		LocalTarget:           "pkg/api/client.go",
		RemoteTarget:          "openapi.json",
		TotalEndpointsChecked: 15,
		Additive:              false,
		Drifts: []diff.DriftItem{
			{
				Severity:   diff.SeverityBreaking,
				Kind:       diff.DriftMissingParam,
				Service:    "UserService",
				Method:     "CreateUser",
				Endpoint:   "POST /v1/users",
				Param:      "email",
				Expected:   "string (required)",
				Actual:     "missing",
				Message:    "missing required parameter email in Go contract",
				Suggestion: "Add Email string `json:\"email\"` to CreateUserRequest",
			},
			{
				Severity: diff.SeverityBreaking,
				Kind:     diff.DriftTypeMismatch,
				Endpoint: "GET /v1/users/{id}",
				Message:  "type mismatch on path param id",
			},
			{
				Severity:   diff.SeverityNonBreaking,
				Kind:       diff.DriftExtraParam,
				Endpoint:   "GET /v1/users",
				Param:      "limit",
				Message:    "optional parameter limit added",
				Suggestion: "Consider updating OpenAPI spec to include limit",
			},
			{
				Severity: diff.SeverityNonBreaking,
				Kind:     diff.DriftDeprecationMismatch,
				Endpoint: "GET /v1/legacy",
				Message:  "endpoint is deprecated in remote spec",
			},
			{
				Severity: diff.SeverityGhost,
				Kind:     diff.DriftMissingEndpoint,
				Endpoint: "DELETE /v1/old",
				Message:  "endpoint declared locally but missing in remote spec",
			},
		},
	}

	assert.Equal(t, 2, fullReport.BreakingCount())
	assert.Equal(t, 2, fullReport.NonBreakingCount())
	assert.Equal(t, 1, fullReport.GhostCount())
	assert.True(t, fullReport.HasBreaking())
	assert.True(t, fullReport.HasDrift())
	assert.Equal(t, 1, fullReport.ExitCode())

	rendered := fullReport.Render(true)
	assert.Contains(t, rendered, "Breaking Changes (2)")
	assert.Contains(t, rendered, "Expected (Remote): string (required) | Actual (Local): missing")
	assert.Contains(t, rendered, "Suggestion: Add Email string")
	assert.Contains(t, rendered, "Non-Breaking Drifts (2)")
	assert.Contains(t, rendered, "Ghost Endpoints (1)")
	assert.Contains(t, rendered, "Summary: 2 breaking, 2 non-breaking, 1 ghost(s)")

	// 4. Additive rendering with drifts
	fullReport.Additive = true
	renderedAdditive := fullReport.Render(false)
	assert.Contains(t, renderedAdditive, "Additive Mode: Ghost Endpoints Suppressed")
	assert.Contains(t, renderedAdditive, "ghost endpoints ignored in additive mode")

	// 5. Ghost-only report exit code
	ghostOnlyReport := &diff.DiffReport{
		Drifts: []diff.DriftItem{
			{
				Severity: diff.SeverityGhost,
				Kind:     diff.DriftMissingEndpoint,
				Endpoint: "DELETE /v1/old",
			},
		},
	}
	assert.Equal(t, 2, ghostOnlyReport.ExitCode())

	// 6. JSON serialization
	jsonBytes, err := fullReport.JSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"severity": "BREAKING"`)

	renderJSONBytes, err := fullReport.RenderJSON()
	require.NoError(t, err)
	assert.Equal(t, string(jsonBytes), string(renderJSONBytes))

	// 7. DiffOptions
	opts := diff.DiffOptions{Additive: true}
	assert.True(t, opts.Additive)
}
