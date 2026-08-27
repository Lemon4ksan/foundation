// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package casing_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/text/casing"
)

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
