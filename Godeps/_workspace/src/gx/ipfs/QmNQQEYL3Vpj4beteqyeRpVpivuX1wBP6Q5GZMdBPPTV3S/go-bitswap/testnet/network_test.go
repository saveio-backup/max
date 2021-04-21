package bitswap

import (
	"context"
	"sync"
	"testing"

	bsmsg "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNQQEYL3Vpj4beteqyeRpVpivuX1wBP6Q5GZMdBPPTV3S/go-bitswap/message"
	bsnet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNQQEYL3Vpj4beteqyeRpVpivuX1wBP6Q5GZMdBPPTV3S/go-bitswap/network"

	delay "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRJVNatYJwTAHgdSM1Xef9QVQ1Ch3XHdmcrykjP5Y4soL/go-ipfs-delay"
	blocks "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVzK524a2VWLqyvtBeiHKsUAWYgeAk4DBeZoY7vpNPNRx/go-block-format"
	testutil "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXG74iiKQnDstVQq9fPFQEB6JTNSWBbAWE1qsq6L4E5sR/go-testutil"
	mockrouting "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qma19TdQ7W26jbfuPgdo9Zi4qtjks1zeXzX86mtEYWYCiw/go-ipfs-routing/mock"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
)

func TestSendMessageAsyncButWaitForResponse(t *testing.T) {
	net := VirtualNetwork(mockrouting.NewServer(), delay.Fixed(0))
	responderPeer := testutil.RandIdentityOrFatal(t)
	waiter := net.Adapter(testutil.RandIdentityOrFatal(t))
	responder := net.Adapter(responderPeer)

	var wg sync.WaitGroup

	wg.Add(1)

	expectedStr := "received async"

	responder.SetDelegate(lambda(func(
		ctx context.Context,
		fromWaiter peer.ID,
		msgFromWaiter bsmsg.BitSwapMessage) {

		msgToWaiter := bsmsg.New(true)
		msgToWaiter.AddBlock(blocks.NewBlock([]byte(expectedStr)))
		waiter.SendMessage(ctx, fromWaiter, msgToWaiter)
	}))

	waiter.SetDelegate(lambda(func(
		ctx context.Context,
		fromResponder peer.ID,
		msgFromResponder bsmsg.BitSwapMessage) {

		// TODO assert that this came from the correct peer and that the message contents are as expected
		ok := false
		for _, b := range msgFromResponder.Blocks() {
			if string(b.RawData()) == expectedStr {
				wg.Done()
				ok = true
			}
		}

		if !ok {
			t.Fatal("Message not received from the responder")
		}
	}))

	messageSentAsync := bsmsg.New(true)
	messageSentAsync.AddBlock(blocks.NewBlock([]byte("data")))
	errSending := waiter.SendMessage(
		context.Background(), responderPeer.ID(), messageSentAsync)
	if errSending != nil {
		t.Fatal(errSending)
	}

	wg.Wait() // until waiter delegate function is executed
}

type receiverFunc func(ctx context.Context, p peer.ID,
	incoming bsmsg.BitSwapMessage)

// lambda returns a Receiver instance given a receiver function
func lambda(f receiverFunc) bsnet.Receiver {
	return &lambdaImpl{
		f: f,
	}
}

type lambdaImpl struct {
	f func(ctx context.Context, p peer.ID, incoming bsmsg.BitSwapMessage)
}

func (lam *lambdaImpl) ReceiveMessage(ctx context.Context,
	p peer.ID, incoming bsmsg.BitSwapMessage) {
	lam.f(ctx, p, incoming)
}

func (lam *lambdaImpl) ReceiveError(err error) {
	// TODO log error
}

func (lam *lambdaImpl) PeerConnected(p peer.ID) {
	// TODO
}
func (lam *lambdaImpl) PeerDisconnected(peer.ID) {
	// TODO
}
