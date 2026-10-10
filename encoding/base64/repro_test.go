package base64_test

import (
	"math"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/base64"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestRepro_Base64URLEncodedLen_NegativeAndOverflow(t *testing.T) {
	t.Run("negative length returns 0", func(t *testing.T) {
		assert.Equal(t, 0, base64.Base64URLEncodedLen(-1))
		assert.Equal(t, 0, base64.Base64URLEncodedLen(-100))
	})

	t.Run("overflow panics", func(t *testing.T) {
		defer func() {
			r := recover()
			require.NotNil(t, r)
		}()
		_ = base64.Base64URLEncodedLen(math.MaxInt / 4)
	})
}
