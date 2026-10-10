package svcb_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/net/dns/svcb"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestRepro_MarshalRDATA_OversizedParam(t *testing.T) {
	rec := &svcb.Record{
		Priority:   1,
		TargetName: ".",
		Params: map[svcb.SvcParamKey][]byte{
			svcb.ParamECH: make([]byte, 70000),
		},
	}
	_, err := rec.MarshalRDATA()
	assert.ErrorIs(t, err, svcb.ErrMalformedParam)
}

func TestRepro_ParseRDATA_MandatoryKeyValidation(t *testing.T) {
	t.Run("mandatory listing mandatory (0) rejected", func(t *testing.T) {
		// Priority 1, TargetName ".", Key 0 (mandatory) with value [0, 0]
		data := []byte{
			0x00, 0x01, // Priority 1
			0x00,       // TargetName "."
			0x00, 0x00, // Key 0 (mandatory)
			0x00, 0x02, // Len 2
			0x00, 0x00, // lists key 0 (mandatory)!
		}
		_, err := svcb.ParseRDATA(data)
		assert.ErrorIs(t, err, svcb.ErrMalformedParam)
	})

	t.Run("mandatory unsorted keys rejected", func(t *testing.T) {
		// Priority 1, TargetName ".", Key 0 (mandatory) with value [0, 3, 0, 1] (keys 3, 1 unsorted)
		data := []byte{
			0x00, 0x01, // Priority 1
			0x00,       // TargetName "."
			0x00, 0x00, // Key 0 (mandatory)
			0x00, 0x04, // Len 4
			0x00, 0x03, 0x00, 0x01, // keys 3, 1
		}
		_, err := svcb.ParseRDATA(data)
		assert.ErrorIs(t, err, svcb.ErrUnsortedParamKeys)
	})

	t.Run("mandatory duplicate keys rejected", func(t *testing.T) {
		// Priority 1, TargetName ".", Key 0 (mandatory) with value [0, 1, 0, 1] (duplicate key 1)
		data := []byte{
			0x00, 0x01, // Priority 1
			0x00,       // TargetName "."
			0x00, 0x00, // Key 0 (mandatory)
			0x00, 0x04, // Len 4
			0x00, 0x01, 0x00, 0x01, // keys 1, 1
		}
		_, err := svcb.ParseRDATA(data)
		assert.ErrorIs(t, err, svcb.ErrDuplicateParamKey)
	})
}

func TestRepro_ParseRDATA_InvalidHintLengths(t *testing.T) {
	t.Run("ipv4hint with 3 bytes rejected", func(t *testing.T) {
		data := []byte{
			0x00, 0x01, // Priority 1
			0x00,       // TargetName "."
			0x00, 0x04, // Key 4 (ipv4hint)
			0x00, 0x03, // Len 3 (invalid!)
			1, 2, 3,
		}
		_, err := svcb.ParseRDATA(data)
		assert.ErrorIs(t, err, svcb.ErrMalformedParam)
	})

	t.Run("ipv4hint with 5 bytes rejected", func(t *testing.T) {
		data := []byte{
			0x00, 0x01, // Priority 1
			0x00,       // TargetName "."
			0x00, 0x04, // Key 4 (ipv4hint)
			0x00, 0x05, // Len 5 (invalid!)
			1, 2, 3, 4, 5,
		}
		_, err := svcb.ParseRDATA(data)
		assert.ErrorIs(t, err, svcb.ErrMalformedParam)
	})

	t.Run("ipv6hint with 15 bytes rejected", func(t *testing.T) {
		data := []byte{
			0x00, 0x01, // Priority 1
			0x00,       // TargetName "."
			0x00, 0x06, // Key 6 (ipv6hint)
			0x00, 0x0f, // Len 15 (invalid!)
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
		}
		_, err := svcb.ParseRDATA(data)
		assert.ErrorIs(t, err, svcb.ErrMalformedParam)
	})
}

func TestRepro_ParseRDATA_MalformedALPN(t *testing.T) {
	t.Run("alpn with 0 length prefix string rejected", func(t *testing.T) {
		// ALPN containing length prefix 0
		data := []byte{
			0x00, 0x01, // Priority 1
			0x00,       // TargetName "."
			0x00, 0x01, // Key 1 (alpn)
			0x00, 0x01, // Len 1
			0x00, // length prefix 0 (empty alpn string is invalid)
		}
		_, err := svcb.ParseRDATA(data)
		assert.ErrorIs(t, err, svcb.ErrMalformedParam)
	})

	t.Run("alpn with truncated string length prefix rejected", func(t *testing.T) {
		// ALPN claiming 5 bytes but only 2 provided
		data := []byte{
			0x00, 0x01, // Priority 1
			0x00,       // TargetName "."
			0x00, 0x01, // Key 1 (alpn)
			0x00, 0x03, // Len 3
			0x05, 'h', '2', // length prefix 5 > remaining 2
		}
		_, err := svcb.ParseRDATA(data)
		assert.ErrorIs(t, err, svcb.ErrMalformedParam)
	})
}
