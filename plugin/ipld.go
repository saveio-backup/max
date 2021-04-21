package plugin

import (
	"github.com/saveio/max/core/coredag"

	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

// PluginIPLD is an interface that can be implemented to add handlers for
// for different IPLD formats
type PluginIPLD interface {
	Plugin

	RegisterBlockDecoders(dec ipld.BlockDecoder) error
	RegisterInputEncParsers(iec coredag.InputEncParsers) error
}
