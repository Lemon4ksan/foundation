package wire_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/net/dns/wire"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestRepro_SkipDomainName_NegativeOffset(t *testing.T) {
	msg := []byte{0x03, 'f', 'o', 'o', 0x00}
	_, err := wire.SkipDomainName(msg, -1)
	assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)

	_, err = wire.SkipDomainName(msg, -999)
	assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)
}
