package unixfs

import (
	cmds "github.com/saveio/max/commands"
	e "github.com/saveio/max/core/commands/e"

	"gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"
)

var UnixFSCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Interact with ONT-IPFS objects representing Unix filesystems.",
		ShortDescription: `
'ont-ipfs file' provides a familiar interface to file systems represented
by ONT-IPFS objects, which hides ont-ipfs implementation details like layout
objects (e.g. fanout and chunking).
`,
		LongDescription: `
'ont-ipfs file' provides a familiar interface to file systems represented
by ONT-IPFS objects, which hides ont-ipfs implementation details like layout
objects (e.g. fanout and chunking).
`,
	},

	Subcommands: map[string]*cmds.Command{
		"ls": LsCmd,
	},
}

// copy+pasted from ../commands.go
func unwrapOutput(i interface{}) (interface{}, error) {
	var (
		ch <-chan interface{}
		ok bool
	)

	if ch, ok = i.(<-chan interface{}); !ok {
		return nil, e.TypeErr(ch, i)
	}

	return <-ch, nil
}
