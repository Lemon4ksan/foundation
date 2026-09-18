package urlkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/net/urlkit"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestURLView_Hostname_IPv6Zone(t *testing.T) {
	v, err := urlkit.ParseView("http://[fe80::1%25eth0]:8080/foo")
	assert.NoError(t, err)
	assert.Equal(t, "fe80::1%eth0", v.Hostname())

	v2, err := urlkit.ParseView("http://[fe80::2]/foo")
	assert.NoError(t, err)
	assert.Equal(t, "fe80::2", v2.Hostname())
}

func TestURLView_ParseView_InvalidHost(t *testing.T) {
	_, err := urlkit.ParseView("http://invalid host.com/foo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")

	_, err2 := urlkit.ParseView("http://foo.com^/foo")
	assert.Error(t, err2)
}

func TestUnescapeMode(t *testing.T) {
	query, err := urlkit.QueryUnescape("foo+bar%20baz")
	assert.NoError(t, err)
	assert.Equal(t, "foo bar baz", query)

	path, err := urlkit.PathUnescape("foo+bar%20baz")
	assert.NoError(t, err)
	assert.Equal(t, "foo+bar baz", path)
}
