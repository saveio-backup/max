package commands

import (
	"io"
	"strings"

	cmds "github.com/saveio/max/commands"
	core "github.com/saveio/max/core"
	e "github.com/saveio/max/core/commands/e"
	"github.com/saveio/max/core/coreunix"
	dag "github.com/saveio/max/merkledag"
	path "github.com/saveio/max/path"
	tar "github.com/saveio/max/tar"

	"gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"
)

var TarCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Utility functions for tar files in ont-ipfs.",
	},

	Subcommands: map[string]*cmds.Command{
		"add": tarAddCmd,
		"cat": tarCatCmd,
	},
}

var tarAddCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Import a tar file into ont-ipfs.",
		ShortDescription: `
'ont-ipfs tar add' will parse a tar file and create a merkledag structure to
represent it.
`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.FileArg("file", true, false, "Tar file to add.").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		fi, err := req.Files().NextFile()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		node, err := tar.ImportTar(req.Context(), fi, nd.DAG)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		c := node.Cid()

		fi.FileName()
		res.SetOutput(&coreunix.AddedObject{
			Name: fi.FileName(),
			Hash: c.String(),
		})
	},
	Type: coreunix.AddedObject{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			v, err := unwrapOutput(res.Output())
			if err != nil {
				return nil, err
			}

			o, ok := v.(*coreunix.AddedObject)
			if !ok {
				return nil, e.TypeErr(o, v)
			}
			return strings.NewReader(o.Hash + "\n"), nil
		},
	},
}

var tarCatCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Export a tar file from ONT-IPFS.",
		ShortDescription: `
'ont-ipfs tar cat' will export a tar file from a previously imported one in ONT-IPFS.
`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("path", true, false, "ont-ipfs path of archive to export.").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		p, err := path.ParsePath(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		root, err := core.Resolve(req.Context(), nd.Namesys, nd.Resolver, p)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		rootpb, ok := root.(*dag.ProtoNode)
		if !ok {
			res.SetError(dag.ErrNotProtobuf, cmdkit.ErrNormal)
			return
		}

		r, err := tar.ExportTar(req.Context(), rootpb, nd.DAG)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		res.SetOutput(r)
	},
}
