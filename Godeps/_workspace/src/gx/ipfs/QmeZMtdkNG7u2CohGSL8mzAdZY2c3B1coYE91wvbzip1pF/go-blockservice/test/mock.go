package bstest

import (
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmeZMtdkNG7u2CohGSL8mzAdZY2c3B1coYE91wvbzip1pF/go-blockservice"

	bitswap "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNQQEYL3Vpj4beteqyeRpVpivuX1wBP6Q5GZMdBPPTV3S/go-bitswap"
	tn "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNQQEYL3Vpj4beteqyeRpVpivuX1wBP6Q5GZMdBPPTV3S/go-bitswap/testnet"
	delay "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRJVNatYJwTAHgdSM1Xef9QVQ1Ch3XHdmcrykjP5Y4soL/go-ipfs-delay"
	mockrouting "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qma19TdQ7W26jbfuPgdo9Zi4qtjks1zeXzX86mtEYWYCiw/go-ipfs-routing/mock"
)

// Mocks returns |n| connected mock Blockservices
func Mocks(n int) []BlockService {
	net := tn.VirtualNetwork(mockrouting.NewServer(), delay.Fixed(0))
	sg := bitswap.NewTestSessionGenerator(net)

	instances := sg.Instances(n)

	var servs []BlockService
	for _, i := range instances {
		servs = append(servs, New(i.Blockstore(), i.Exchange))
	}
	return servs
}
