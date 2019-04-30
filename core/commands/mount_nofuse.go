// +build linux darwin freebsd netbsd openbsd
// +build nofuse

package commands

import (
	cmds "github.com/saveio/max/commands"

	"gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"
)

var MountCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Mounts ont-ipfs to the filesystem (disabled).",
		ShortDescription: `
This version of ont-ipfs is compiled without fuse support, which is required
for mounting. If you'd like to be able to mount, please use a version of
ipfs compiled with fuse.

For the latest instructions, please check the project's repository:
  http://github.com/saveio/ont-ipfs
`,
	},
}
