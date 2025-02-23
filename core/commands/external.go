package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	cmds "github.com/saveio/max/commands"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"
)

func ExternalBinary() *cmds.Command {
	return &cmds.Command{
		Arguments: []cmdkit.Argument{
			cmdkit.StringArg("args", false, true, "Arguments for subcommand."),
		},
		External: true,
		Run: func(req cmds.Request, res cmds.Response) {
			binname := strings.Join(append([]string{"ont-ipfs"}, req.Path()...), "-")
			_, err := exec.LookPath(binname)
			if err != nil {
				// special case for '--help' on uninstalled binaries.
				for _, arg := range req.Arguments() {
					if arg == "--help" || arg == "-h" {
						buf := new(bytes.Buffer)
						fmt.Fprintf(buf, "%s is an 'external' command.\n", binname)
						fmt.Fprintf(buf, "It does not currently appear to be installed.\n")
						fmt.Fprintf(buf, "Please refer to the ont-ipfs documentation for instructions.\n")
						res.SetOutput(buf)
						return
					}
				}

				res.SetError(fmt.Errorf("%s not installed", binname), cmdkit.ErrNormal)
				return
			}

			r, w := io.Pipe()

			cmd := exec.Command(binname, req.Arguments()...)

			// TODO: make commands lib be able to pass stdin through daemon
			//cmd.Stdin = req.Stdin()
			cmd.Stdin = io.LimitReader(nil, 0)
			cmd.Stdout = w
			cmd.Stderr = w

			// setup env of child program
			env := os.Environ()

			nd, err := req.InvocContext().GetNode()
			if err == nil {
				env = append(env, fmt.Sprintf("IPFS_ONLINE=%t", nd.OnlineMode()))
			}

			cmd.Env = env

			err = cmd.Start()
			if err != nil {
				res.SetError(fmt.Errorf("failed to start subcommand: %s", err), cmdkit.ErrNormal)
				return
			}

			res.SetOutput(r)

			go func() {
				err = cmd.Wait()
				if err != nil {
					res.SetError(err, cmdkit.ErrNormal)
				}

				w.Close()
			}()
		},
	}
}
