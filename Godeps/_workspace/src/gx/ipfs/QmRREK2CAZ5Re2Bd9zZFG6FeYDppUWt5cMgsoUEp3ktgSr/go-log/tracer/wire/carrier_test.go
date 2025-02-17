package wire_test

import (
	"testing"

	tracer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log/tracer"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log/tracer/wire"
)

func TestProtobufCarrier(t *testing.T) {
	var carrier tracer.DelegatingCarrier = &wire.ProtobufCarrier{}

	var traceID, spanID uint64 = 1, 2
	sampled := true
	baggageKey, expVal := "key1", "val1"

	carrier.SetState(traceID, spanID, sampled)
	carrier.SetBaggageItem(baggageKey, expVal)
	gotTraceID, gotSpanID, gotSampled := carrier.State()
	if traceID != gotTraceID || spanID != gotSpanID || sampled != gotSampled {
		t.Errorf("Wanted state %d %d %t, got %d %d %t", spanID, traceID, sampled,
			gotTraceID, gotSpanID, gotSampled)
	}

	gotBaggage := map[string]string{}
	f := func(k, v string) {
		gotBaggage[k] = v
	}

	carrier.GetBaggage(f)
	value, ok := gotBaggage[baggageKey]
	if !ok {
		t.Errorf("Expected baggage item %s to exist", baggageKey)
	}
	if value != expVal {
		t.Errorf("Expected key %s to be %s, got %s", baggageKey, expVal, value)
	}
}
