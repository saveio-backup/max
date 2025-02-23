package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	cmds "github.com/saveio/max/commands"
	core "github.com/saveio/max/core"

	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"
)

// P2PListenerInfoOutput is output type of ls command
type P2PListenerInfoOutput struct {
	Protocol string
	Address  string
}

// P2PStreamInfoOutput is output type of streams command
type P2PStreamInfoOutput struct {
	HandlerID     string
	Protocol      string
	LocalPeer     string
	LocalAddress  string
	RemotePeer    string
	RemoteAddress string
}

// P2PLsOutput is output type of ls command
type P2PLsOutput struct {
	Listeners []P2PListenerInfoOutput
}

// P2PStreamsOutput is output type of streams command
type P2PStreamsOutput struct {
	Streams []P2PStreamInfoOutput
}

// P2PCmd is the 'ipfs p2p' command
var P2PCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Libp2p stream mounting.",
		ShortDescription: `
Create and use tunnels to remote peers over libp2p

Note: this command is experimental and subject to change as usecases and APIs
are refined`,
	},

	Subcommands: map[string]*cmds.Command{
		"listener": p2pListenerCmd,
		"stream":   p2pStreamCmd,
	},
}

// p2pListenerCmd is the 'ipfs p2p listener' command
var p2pListenerCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline:          "P2P listener management.",
		ShortDescription: "Create and manage listener p2p endpoints",
	},

	Subcommands: map[string]*cmds.Command{
		"ls":    p2pListenerLsCmd,
		"open":  p2pListenerListenCmd,
		"close": p2pListenerCloseCmd,
	},
}

// p2pStreamCmd is the 'ipfs p2p stream' command
var p2pStreamCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline:          "P2P stream management.",
		ShortDescription: "Create and manage p2p streams",
	},

	Subcommands: map[string]*cmds.Command{
		"ls":    p2pStreamLsCmd,
		"dial":  p2pStreamDialCmd,
		"close": p2pStreamCloseCmd,
	},
}

var p2pListenerLsCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "List active p2p listeners.",
	},
	Options: []cmdkit.Option{
		cmdkit.BoolOption("headers", "v", "Print table headers (HandlerID, Protocol, Local, Remote)."),
	},
	Run: func(req cmds.Request, res cmds.Response) {

		n, err := getNode(req)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		output := &P2PLsOutput{}

		for _, listener := range n.P2P.Listeners.Listeners {
			output.Listeners = append(output.Listeners, P2PListenerInfoOutput{
				Protocol: listener.Protocol,
				Address:  listener.Address.String(),
			})
		}

		res.SetOutput(output)
	},
	Type: P2PLsOutput{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			v, err := unwrapOutput(res.Output())
			if err != nil {
				return nil, err
			}

			headers, _, _ := res.Request().Option("headers").Bool()
			list := v.(*P2PLsOutput)
			buf := new(bytes.Buffer)
			w := tabwriter.NewWriter(buf, 1, 2, 1, ' ', 0)
			for _, listener := range list.Listeners {
				if headers {
					fmt.Fprintln(w, "Address\tProtocol")
				}

				fmt.Fprintf(w, "%s\t%s\n", listener.Address, listener.Protocol)
			}
			w.Flush()

			return buf, nil
		},
	},
}

var p2pStreamLsCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "List active p2p streams.",
	},
	Options: []cmdkit.Option{
		cmdkit.BoolOption("headers", "v", "Print table headers (HagndlerID, Protocol, Local, Remote)."),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := getNode(req)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		output := &P2PStreamsOutput{}

		for _, s := range n.P2P.Streams.Streams {
			output.Streams = append(output.Streams, P2PStreamInfoOutput{
				HandlerID: strconv.FormatUint(s.HandlerID, 10),

				Protocol: s.Protocol,

				LocalPeer:    s.LocalPeer.Pretty(),
				LocalAddress: s.LocalAddr.String(),

				RemotePeer:    s.RemotePeer.Pretty(),
				RemoteAddress: s.RemoteAddr.String(),
			})
		}

		res.SetOutput(output)
	},
	Type: P2PStreamsOutput{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			v, err := unwrapOutput(res.Output())
			if err != nil {
				return nil, err
			}

			headers, _, _ := res.Request().Option("headers").Bool()
			list := v.(*P2PStreamsOutput)
			buf := new(bytes.Buffer)
			w := tabwriter.NewWriter(buf, 1, 2, 1, ' ', 0)
			for _, stream := range list.Streams {
				if headers {
					fmt.Fprintln(w, "HandlerID\tProtocol\tLocal\tRemote")
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", stream.HandlerID, stream.Protocol, stream.LocalAddress, stream.RemotePeer)
			}
			w.Flush()

			return buf, nil
		},
	},
}

var p2pListenerListenCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Forward p2p connections to a network multiaddr.",
		ShortDescription: `
Register a p2p connection handler and forward the connections to a specified
address.

Note that the connections originate from the ont-ipfs daemon process.
		`,
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("Protocol", true, false, "Protocol identifier."),
		cmdkit.StringArg("Address", true, false, "Request handling application address."),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := getNode(req)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		proto := "/p2p/" + req.Arguments()[0]
		if n.P2P.CheckProtoExists(proto) {
			res.SetError(errors.New("protocol handler already registered"), cmdkit.ErrNormal)
			return
		}

		addr, err := ma.NewMultiaddr(req.Arguments()[1])
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		_, err = n.P2P.NewListener(n.Context(), proto, addr)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		// Successful response.
		res.SetOutput(&P2PListenerInfoOutput{
			Protocol: proto,
			Address:  addr.String(),
		})
	},
}

var p2pStreamDialCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Dial to a p2p listener.",

		ShortDescription: `
Establish a new connection to a peer service.

When a connection is made to a peer service the ont-ipfs daemon will setup one
time TCP listener and return it's bind port, this way a dialing application
can transparently connect to a p2p service.
		`,
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("Peer", true, false, "Remote peer to connect to"),
		cmdkit.StringArg("Protocol", true, false, "Protocol identifier."),
		cmdkit.StringArg("BindAddress", false, false, "Address to listen for connection/s (default: /ip4/127.0.0.1/tcp/0)."),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := getNode(req)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		addr, peer, err := ParsePeerParam(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		proto := "/p2p/" + req.Arguments()[1]

		bindAddr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")
		if len(req.Arguments()) == 3 {
			bindAddr, err = ma.NewMultiaddr(req.Arguments()[2])
			if err != nil {
				res.SetError(err, cmdkit.ErrNormal)
				return
			}
		}

		listenerInfo, err := n.P2P.Dial(n.Context(), addr, peer, proto, bindAddr)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		output := P2PListenerInfoOutput{
			Protocol: listenerInfo.Protocol,
			Address:  listenerInfo.Address.String(),
		}

		res.SetOutput(&output)
	},
}

var p2pListenerCloseCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Close active p2p listener.",
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("Protocol", false, false, "P2P listener protocol"),
	},
	Options: []cmdkit.Option{
		cmdkit.BoolOption("all", "a", "Close all listeners."),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		res.SetOutput(nil)

		n, err := getNode(req)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		closeAll, _, _ := req.Option("all").Bool()
		var proto string

		if !closeAll {
			if len(req.Arguments()) == 0 {
				res.SetError(errors.New("no protocol name specified"), cmdkit.ErrNormal)
				return
			}

			proto = "/p2p/" + req.Arguments()[0]
		}

		for _, listener := range n.P2P.Listeners.Listeners {
			if !closeAll && listener.Protocol != proto {
				continue
			}
			listener.Close()
			if !closeAll {
				break
			}
		}
	},
}

var p2pStreamCloseCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Close active p2p stream.",
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("HandlerID", false, false, "Stream HandlerID"),
	},
	Options: []cmdkit.Option{
		cmdkit.BoolOption("all", "a", "Close all streams."),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		res.SetOutput(nil)

		n, err := getNode(req)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		closeAll, _, _ := req.Option("all").Bool()
		var handlerID uint64

		if !closeAll {
			if len(req.Arguments()) == 0 {
				res.SetError(errors.New("no HandlerID specified"), cmdkit.ErrNormal)
				return
			}

			handlerID, err = strconv.ParseUint(req.Arguments()[0], 10, 64)
			if err != nil {
				res.SetError(err, cmdkit.ErrNormal)
				return
			}
		}

		for _, stream := range n.P2P.Streams.Streams {
			if !closeAll && handlerID != stream.HandlerID {
				continue
			}
			stream.Close()
			if !closeAll {
				break
			}
		}
	},
}

func getNode(req cmds.Request) (*core.IpfsNode, error) {
	n, err := req.InvocContext().GetNode()
	if err != nil {
		return nil, err
	}

	config, err := n.Repo.Config()
	if err != nil {
		return nil, err
	}

	if !config.Experimental.Libp2pStreamMounting {
		return nil, errors.New("libp2p stream mounting not enabled")
	}

	if !n.OnlineMode() {
		return nil, errNotOnline
	}

	return n, nil
}
