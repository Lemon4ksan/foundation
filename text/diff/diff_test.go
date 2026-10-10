package diff_test

// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
	"github.com/lemon4ksan/foundation/text/diff"
)

func TestAdversarial_NilAndEmptyDrifts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		report   *diff.Report
		additive bool
	}{
		{
			name: "NilDrifts_Standard",
			report: &diff.Report{
				LocalTarget:           "local.go",
				RemoteTarget:          "spec.json",
				TotalEndpointsChecked: 0,
				Additive:              false,
				Drifts:                nil,
			},
			additive: false,
		},
		{
			name: "EmptySliceDrifts_Standard",
			report: &diff.Report{
				LocalTarget:           "local.go",
				RemoteTarget:          "spec.json",
				TotalEndpointsChecked: 10,
				Additive:              false,
				Drifts:                []diff.Drift{},
			},
			additive: false,
		},
		{
			name: "NilDrifts_Additive",
			report: &diff.Report{
				LocalTarget:           "local.go",
				RemoteTarget:          "spec.json",
				TotalEndpointsChecked: 5,
				Additive:              true,
				Drifts:                nil,
			},
			additive: true,
		},
		{
			name: "EmptySliceDrifts_Additive",
			report: &diff.Report{
				LocalTarget:           "local.go",
				RemoteTarget:          "spec.json",
				TotalEndpointsChecked: 5,
				Additive:              true,
				Drifts:                []diff.Drift{},
			},
			additive: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := tc.report

			assert.Equal(t, 0, r.BreakingCount())
			assert.Equal(t, 0, r.NonBreakingCount())
			assert.Equal(t, 0, r.GhostCount())
			assert.False(t, r.HasBreaking())
			assert.False(t, r.HasDrift())
			assert.Equal(t, 0, r.ExitCode())

			renderedNoColor := r.Render(false)
			renderedColor := r.Render(true)
			assert.NotEmpty(t, renderedNoColor)
			assert.NotEmpty(t, renderedColor)

			if tc.additive {
				assert.Contains(t, renderedNoColor, "100% satisfied by local contract")
			} else {
				assert.Contains(t, renderedNoColor, "100% in sync with OpenAPI specification")
			}

			// JSON Serialization
			jsonBytes, err := r.JSON()
			require.NoError(t, err)
			assert.NotEmpty(t, jsonBytes)

			renderJSONBytes, err := r.RenderJSON()
			require.NoError(t, err)
			assert.Equal(t, string(jsonBytes), string(renderJSONBytes))

			// Round-trip unmarshal
			var unmarshaled diff.Report
			err = json.Unmarshal(jsonBytes, &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, r.LocalTarget, unmarshaled.LocalTarget)
			assert.Equal(t, r.RemoteTarget, unmarshaled.RemoteTarget)
			assert.Equal(t, r.TotalEndpointsChecked, unmarshaled.TotalEndpointsChecked)
			assert.Equal(t, r.Additive, unmarshaled.Additive)
			assert.Equal(t, 0, len(unmarshaled.Drifts))
		})
	}
}

func TestAdversarial_ExtremeDriftsVolume(t *testing.T) {
	t.Parallel()

	const count = 30_000
	drifts := make([]diff.Drift, 0, count)

	var (
		expectedBreaking    int
		expectedNonBreaking int
		expectedGhost       int
	)

	for i := range count {
		var sev diff.DriftSeverity
		switch i % 4 {
		case 0:
			sev = diff.SeverityBreaking
			expectedBreaking++
		case 1:
			sev = diff.SeverityNonBreaking
			expectedNonBreaking++
		case 2:
			sev = diff.SeverityGhost
			expectedGhost++
		case 3:
			// Unrecognized / custom severity
			sev = diff.DriftSeverity("CUSTOM_SEVERITY")
		}

		drifts = append(drifts, diff.Drift{
			Severity:   sev,
			Kind:       diff.DriftMissingParam,
			Service:    fmt.Sprintf("Service-%d", i),
			Method:     fmt.Sprintf("Method-%d", i),
			Endpoint:   fmt.Sprintf("/v1/resource/%d", i),
			Param:      fmt.Sprintf("param_%d", i),
			Expected:   "string",
			Actual:     "int",
			Message:    fmt.Sprintf("Discrepancy at index %d", i),
			Suggestion: fmt.Sprintf("Fix parameter %d", i),
		})
	}

	report := &diff.Report{
		LocalTarget:           "generated/client.go",
		RemoteTarget:          "spec/huge_openapi.json",
		TotalEndpointsChecked: count,
		Additive:              false,
		Drifts:                drifts,
	}

	assert.Equal(t, expectedBreaking, report.BreakingCount())
	assert.Equal(t, expectedNonBreaking, report.NonBreakingCount())
	assert.Equal(t, expectedGhost, report.GhostCount())
	assert.True(t, report.HasBreaking())
	assert.True(t, report.HasDrift())
	assert.Equal(t, 1, report.ExitCode()) // Breaking takes precedence

	// Render without crashing or out of memory
	rendered := report.Render(false)
	assert.Contains(t, rendered, fmt.Sprintf("Breaking Changes (%d)", expectedBreaking))
	assert.Contains(t, rendered, fmt.Sprintf("Non-Breaking Drifts (%d)", expectedNonBreaking))
	assert.Contains(t, rendered, fmt.Sprintf("Ghost Endpoints (%d)", expectedGhost))

	// JSON serialization
	jsonBytes, err := report.JSON()
	require.NoError(t, err)

	var unmarshaled diff.Report
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, count, len(unmarshaled.Drifts))
	assert.Equal(t, expectedBreaking, unmarshaled.BreakingCount())
	assert.Equal(t, expectedNonBreaking, unmarshaled.NonBreakingCount())
	assert.Equal(t, expectedGhost, unmarshaled.GhostCount())
}

func TestAdversarial_MalformedAndEdgeCaseDriftFields(t *testing.T) {
	t.Parallel()

	longString := strings.Repeat("A", 10_000)
	unicodeString := "🔥 測試 тест 🚀 \u0000 \t \n \r"

	report := &diff.Report{
		LocalTarget:           longString,
		RemoteTarget:          unicodeString,
		TotalEndpointsChecked: -100, // Negative endpoints checked
		Additive:              true,
		Drifts: []diff.Drift{
			{
				// Entirely empty drift
				Severity: diff.SeverityBreaking,
			},
			{
				// Only Expected, no Actual
				Severity: diff.SeverityBreaking,
				Endpoint: "/test/expected-only",
				Expected: "remote_model",
				Actual:   "",
			},
			{
				// Only Actual, no Expected
				Severity: diff.SeverityBreaking,
				Endpoint: "/test/actual-only",
				Expected: "",
				Actual:   "local_model",
			},
			{
				// Unicode and long values
				Severity:   diff.SeverityNonBreaking,
				Kind:       diff.DriftTypeMismatch,
				Endpoint:   unicodeString,
				Message:    longString,
				Suggestion: unicodeString,
			},
			{
				// Ghost with weird severity casing
				Severity: diff.SeverityGhost,
				Kind:     diff.DriftMissingEndpoint,
				Endpoint: "/ghost/path",
				Message:  "ghost message",
			},
		},
	}

	assert.Equal(t, 3, report.BreakingCount())
	assert.Equal(t, 1, report.NonBreakingCount())
	assert.Equal(t, 1, report.GhostCount())
	assert.True(t, report.HasBreaking())
	assert.True(t, report.HasDrift())

	rendered := report.Render(true)
	assert.NotEmpty(t, rendered)
	assert.Contains(t, rendered, "Additive Mode: Ghost Endpoints Suppressed")

	jsonBytes, err := report.JSON()
	require.NoError(t, err)

	var unmarshaled diff.Report
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, 5, len(unmarshaled.Drifts))
}

func TestAdversarial_ConcurrentReportAccess(t *testing.T) {
	t.Parallel()

	report := &diff.Report{
		LocalTarget:           "api/local.go",
		RemoteTarget:          "api/remote.json",
		TotalEndpointsChecked: 42,
		Additive:              false,
		Drifts: []diff.Drift{
			{
				Severity: diff.SeverityBreaking,
				Kind:     diff.DriftMissingParam,
				Endpoint: "/users",
				Message:  "missing param",
			},
			{
				Severity: diff.SeverityNonBreaking,
				Kind:     diff.DriftExtraParam,
				Endpoint: "/users",
				Message:  "extra param",
			},
			{
				Severity: diff.SeverityGhost,
				Kind:     diff.DriftMissingEndpoint,
				Endpoint: "/legacy",
				Message:  "ghost endpoint",
			},
		},
	}

	const goroutines = 50
	const iterations = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(gid int) {
			defer wg.Done()
			for range iterations {
				assert.Equal(t, 1, report.BreakingCount())
				assert.Equal(t, 1, report.NonBreakingCount())
				assert.Equal(t, 1, report.GhostCount())
				assert.True(t, report.HasBreaking())
				assert.True(t, report.HasDrift())
				assert.Equal(t, 1, report.ExitCode())

				rendered := report.Render(gid%2 == 0)
				assert.NotEmpty(t, rendered)

				b, err := report.JSON()
				assert.NoError(t, err)
				assert.NotEmpty(t, b)

				rb, err := report.RenderJSON()
				assert.NoError(t, err)
				assert.NotEmpty(t, rb)
			}
		}(g)
	}

	wg.Wait()
}
