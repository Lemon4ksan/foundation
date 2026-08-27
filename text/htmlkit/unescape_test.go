// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/text/htmlkit"
)

func TestUnescape(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []byte("Hello World"), htmlkit.Unescape([]byte("Hello World")))
	assert.Equal(t, []byte(`"Hello" & <World>`), htmlkit.Unescape([]byte("&quot;Hello&quot; &amp; &lt;World&gt;")))
	assert.Equal(t, []byte("Price: 100€"), htmlkit.Unescape([]byte("Price: 100&euro;")))
	assert.Equal(t, []byte("Emoji: 😀"), htmlkit.Unescape([]byte("Emoji: &#x1F600;")))
	assert.Equal(t, []byte("Char: A"), htmlkit.Unescape([]byte("Char: &#65;")))
}
