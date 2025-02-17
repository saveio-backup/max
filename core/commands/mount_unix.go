// +build linux darwin freebsd netbsd openbsd
// +build !nofuse

package commands

import (
	"fmt"
	"io"
	"strings"

	cmds "github.com/saveio/max/commands"
	e "github.com/saveio/max/core/commands/e"
	nodeMount "github.com/saveio/max/fuse/node"
	config "github.com/saveio/max/repo/config"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"
)

var MountCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Mounts ONT-IPFS to the filesystem (read-only).",
		ShortDescription: `
Mount ONT-IPFS at a read-only mountpoint on the OS (default: /ipfs and /ipns).
All ONT-IPFS objects will be accessible under that directory. Note that the
root will not be listable, as it is virtual. Access known paths directly.

You may have to create /ipfs and /ipns before using 'ipfs mount':

> sudo mkdir /ipfs /ipns
> sudo chown $(whoami) /ipfs /ipns
> ont-ipfs daemon &
> ont-ipfs mount
`,
		LongDescription: `
Mount ONT-IPFS at a read-only mountpoint on the OS. The default, /ipfs and /ipns,
are set in the configuration file, but can be overriden by the options.
All ONT-IPFS objects will be accessible under this directory. Note that the
root will not be listable, as it is virtual. Access known paths directly.

You may have to create /ipfs and /ipns before using 'ipfs mount':

> sudo mkdir /ipfs /ipns
> sudo chown $(whoami) /ipfs /ipns
> ont-ipfs daemon &
> ont-ipfs mount

Example:

# setup
> mkdir foo
> echo "baz" > foo/bar
> ont-ipfs add -r foo
added QmWLdkp93sNxGRjnFHPaYg8tCQ35NBY3XPn6KiETd3Z4WR foo/bar
added QmSh5e7S6fdcu75LAbXNZAFY2nGyZUJXyLCJDvn2zRkWyC foo
> ont-ipfs ls QmSh5e7S6fdcu75LAbXNZAFY2nGyZUJXyLCJDvn2zRkWyC
QmWLdkp93sNxGRjnFHPaYg8tCQ35NBY3XPn6KiETd3Z4WR 12 bar
> ont-ipfs cat QmWLdkp93sNxGRjnFHPaYg8tCQ35NBY3XPn6KiETd3Z4WR
baz

# mount
> ont-ipfs daemon &
> ont-ipfs mount
IPFS mounted at: /ipfs
IPNS mounted at: /ipns
> cd /ipfs/QmSh5e7S6fdcu75LAbXNZAFY2nGyZUJXyLCJDvn2zRkWyC
> ls
bar
> cat bar
baz
> cat /ipfs/QmSh5e7S6fdcu75LAbXNZAFY2nGyZUJXyLCJDvn2zRkWyC/bar
baz
> cat /ipfs/QmWLdkp93sNxGRjnFHPaYg8tCQ35NBY3XPn6KiETd3Z4WR
baz
`,
	},
	Options: []cmdkit.Option{
		cmdkit.StringOption("ipfs-path", "f", "The path where ONT-IPFS should be mounted."),
		cmdkit.StringOption("ipns-path", "n", "The path where IPNS should be mounted."),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		cfg, err := req.InvocContext().GetConfig()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		node, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		// error if we aren't running node in online mode
		if node.LocalMode() {
			res.SetError(errNotOnline, cmdkit.ErrClient)
			return
		}

		fsdir, found, err := req.Option("f").String()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}
		if !found {
			fsdir = cfg.Mounts.IPFS // use default value
		}

		// get default mount points
		nsdir, found, err := req.Option("n").String()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}
		if !found {
			nsdir = cfg.Mounts.IPNS // NB: be sure to not redeclare!
		}

		err = nodeMount.Mount(node, fsdir, nsdir)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		var output config.Mounts
		output.IPFS = fsdir
		output.IPNS = nsdir
		res.SetOutput(&output)
	},
	Type: config.Mounts{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			v, err := unwrapOutput(res.Output())
			if err != nil {
				return nil, err
			}

			mnts, ok := v.(*config.Mounts)
			if !ok {
				return nil, e.TypeErr(mnts, v)
			}

			s := fmt.Sprintf("IPFS mounted at: %s\n", mnts.IPFS)
			s += fmt.Sprintf("IPNS mounted at: %s\n", mnts.IPNS)
			return strings.NewReader(s), nil
		},
	},
}
