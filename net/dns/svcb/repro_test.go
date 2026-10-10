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
