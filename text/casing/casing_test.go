// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package casing_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"

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
