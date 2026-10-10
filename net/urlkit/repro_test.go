package urlkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/net/urlkit"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestRepro_IPv6HostWithoutBrackets_HostnameAndPort(t *testing.T) {
	// IPv6 address with multiple colons and no brackets (or unclosed) should NOT treat
	// the trailing colon as a port separator.
	v := urlkit.URLView{Host: "2001:db8::1"}
	assert.Equal(t, "2001:db8::1", v.Hostname())
	assert.Equal(t, "", v.Port())
}

func TestRepro_IPv6UnclosedBracket_HostnameAndPort(t *testing.T) {
	// Unclosed IPv6 bracket "[2001:db8::1:8080" should not be parsed as valid bracketed host
	v := urlkit.URLView{Host: "[2001:db8::1"}
	assert.Equal(t, "[2001:db8::1", v.Hostname())
	assert.Equal(t, "", v.Port())
}
