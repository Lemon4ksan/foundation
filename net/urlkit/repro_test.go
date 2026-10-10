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

func TestRepro_CorruptedPercentEncoding(t *testing.T) {
	// Corrupted percent-encoding sequences must return ErrInvalidEscape
	corrupted := []string{
		"%",
		"%a",
		"%ZZ",
		"hello%2",
		"bad%G1",
	}

	for _, s := range corrupted {
		_, err := urlkit.QueryUnescape(s)
		assert.ErrorIs(t, err, urlkit.ErrInvalidEscape)

		_, err = urlkit.PathUnescape(s)
		assert.ErrorIs(t, err, urlkit.ErrInvalidEscape)
	}
}

func TestRepro_NullByteInjection_ControlChars(t *testing.T) {
	// Null byte \x00 in URL must be rejected
	_, err := urlkit.ParseView("http://example.com/admin\x00extra")
	assert.Error(t, err, "expected error on null byte in URL")

	// Host containing % must be rejected unless valid IPv6 zone
	_, err = urlkit.ParseView("http://example%00com/")
	assert.Error(t, err, "expected error on percent in non-IPv6 host")
}

func TestRepro_IPv6_ScopedZoneID(t *testing.T) {
	// Literal zone ID [fe80::1%eth0]:8080
	u, err := urlkit.Parse("http://[fe80::1%eth0]:8080/test")
	assert.NoError(t, err)
	assert.Equal(t, "fe80::1%eth0", u.Hostname())
	assert.Equal(t, "8080", u.Port())

	// Percent-encoded zone ID [fe80::1%25eth0]:8080
	u2, err := urlkit.Parse("http://[fe80::1%25eth0]:8080/test")
	assert.NoError(t, err)
	assert.Equal(t, "fe80::1%eth0", u2.Hostname())
	assert.Equal(t, "8080", u2.Port())
}

func TestRepro_Authority_MultipleAtSymbols(t *testing.T) {
	// Multiple unescaped @ in authority is invalid userinfo (SSRF vector)
	_, err := urlkit.ParseView("http://user@attacker.com@host:8080/")
	assert.Error(t, err, "expected error for multiple @ in authority")
}

func TestRepro_Authority_MultipleColons(t *testing.T) {
	// Multiple colons in userinfo represents username:password:with:colons
	v, err := urlkit.ParseView("http://user:p:a:s:s@host:8080/")
	assert.NoError(t, err)
	assert.Equal(t, "user:p:a:s:s", v.User)
	assert.Equal(t, "host:8080", v.Host)

	u := v.ToURL()
	assert.Equal(t, "user", u.User.Username())
	pass, _ := u.User.Password()
	assert.Equal(t, "p:a:s:s", pass)
}
