// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package html_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"

	"github.com/lemon4ksan/foundation/text/html"
)

func TestUnescape(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []byte("Hello World"), html.Unescape([]byte("Hello World")))
	assert.Equal(t, []byte(`"Hello" & <World>`), html.Unescape([]byte("&quot;Hello&quot; &amp; &lt;World&gt;")))
	assert.Equal(t, []byte("Price: 100€"), html.Unescape([]byte("Price: 100&euro;")))
	assert.Equal(t, []byte("Emoji: 😀"), html.Unescape([]byte("Emoji: &#x1F600;")))
	assert.Equal(t, []byte("Char: A"), html.Unescape([]byte("Char: &#65;")))
}
