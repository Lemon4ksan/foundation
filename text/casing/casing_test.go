// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package casing_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/text/casing"
)

func TestCasingConversions(t *testing.T) {
	t.Parallel()

	// ToSnake
	assert.Equal(t, "user_name", casing.ToSnake("userName"))
	assert.Equal(t, "user_name", casing.ToSnake("UserName"))
	assert.Equal(t, "http_server_url", casing.ToSnake("HTTPServerURL"))
	assert.Equal(t, "content_type", casing.ToSnake("content-type"))

	// ToCamel
	assert.Equal(t, "userName", casing.ToCamel("user_name"))
	assert.Equal(t, "userName", casing.ToCamel("User_Name"))
	assert.Equal(t, "httpServerUrl", casing.ToCamel("http_server_url"))

	// ToPascal
	assert.Equal(t, "UserName", casing.ToPascal("user_name"))
	assert.Equal(t, "HttpServerUrl", casing.ToPascal("http_server_url"))

	// ToKebab
	assert.Equal(t, "user-name", casing.ToKebab("userName"))
	assert.Equal(t, "content-type", casing.ToKebab("ContentType"))

	// ToScreamingSnake
	assert.Equal(t, "TIMEOUT_SECONDS", casing.ToScreamingSnake("timeoutSeconds"))
	assert.Equal(t, "HTTP_SERVER_URL", casing.ToScreamingSnake("httpServerURL"))
}

func FuzzCasingConversions(f *testing.F) {
	f.Add("HTTPServerURL")
	f.Add("user_id")
	f.Add("camelCaseWord")
	f.Add("SCREAMING_SNAKE_CASE")
	f.Add("kebab-case-string")
	f.Add("")

	f.Fuzz(func(t *testing.T, s string) {
		_ = casing.ToSnake(s)
		_ = casing.ToCamel(s)
		_ = casing.ToPascal(s)
		_ = casing.ToKebab(s)
		_ = casing.ToScreamingSnake(s)
	})
}

func TestAllocations_ToSnake(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_ = casing.ToSnake("PascalCaseExample")
	})
	if allocs > 0 {
		t.Logf("ToSnake allocated %v times", allocs)
	}
}

func TestCasing_TrimDelimiters(t *testing.T) {
	res := casing.ToSnake("  hello  ")
	assert.Equal(t, "hello", res)

	res2 := casing.ToKebab("a--b")
	assert.Equal(t, "a-b", res2)
}

func TestCasing_Unicode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "мой_тест", casing.ToSnake("МойТест"))
	assert.Equal(t, "привет_мир", casing.ToSnake("ПриветМир"))
	assert.Equal(t, "ПРИВЕТ_МИР", casing.ToScreamingSnake("ПриветМир"))
	assert.Equal(t, "мой-тест", casing.ToKebab("МойТест"))
}

