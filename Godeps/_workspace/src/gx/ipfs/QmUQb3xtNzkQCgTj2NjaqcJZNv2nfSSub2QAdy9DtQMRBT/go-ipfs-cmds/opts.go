package cmds

import (
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmPVqQHEfLpqK7JLCsUkyam7rhuV3MAeZ9gueQQCrBwCta/go-ipfs-cmdkit"
)

// Flag names
const (
	EncShort     = "enc"
	EncLong      = "encoding"
	RecShort     = "r"
	RecLong      = "recursive"
	ChanOpt      = "stream-channels"
	TimeoutOpt   = "timeout"
	OptShortHelp = "h"
	OptLongHelp  = "help"
)

// options that are used by this package
var OptionEncodingType = cmdkit.StringOption(EncLong, EncShort, "The encoding type the output should be encoded with (json, xml, or text)").WithDefault("text")
var OptionRecursivePath = cmdkit.BoolOption(RecLong, RecShort, "Add directory paths recursively").WithDefault(false)
var OptionStreamChannels = cmdkit.BoolOption(ChanOpt, "Stream channel output")
var OptionTimeout = cmdkit.StringOption(TimeoutOpt, "set a global timeout on the command")
