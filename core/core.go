/*
Package core implements the IpfsNode object and related methods.

Packages underneath core/ provide a (relatively) stable, low-level API
to carry out most IPFS-related tasks.  For more details on the other
interfaces and how core/... fits into the bigger ONT-IPFS picture, see:

  $ godoc github.com/saveio/ont-ipfs
*/
package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	bserv "github.com/saveio/max/blockservice"
	bitswap "github.com/saveio/max/exchange/bitswap"
	bsnet "github.com/saveio/max/exchange/bitswap/network"
	rp "github.com/saveio/max/exchange/reprovide"
	filestore "github.com/saveio/max/filestore"
	mount "github.com/saveio/max/fuse/mount"
	merkledag "github.com/saveio/max/merkledag"
	mfs "github.com/saveio/max/mfs"
	namesys "github.com/saveio/max/namesys"
	ipnsrp "github.com/saveio/max/namesys/republisher"
	p2p "github.com/saveio/max/p2p"
	"github.com/saveio/max/path/resolver"
	pin "github.com/saveio/max/pin"
	repo "github.com/saveio/max/repo"
	config "github.com/saveio/max/repo/config"
	ft "github.com/saveio/max/unixfs"

	addrutil "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNSWW3Sb4eju4o2djPQ1L1c2Zj9XN9sMYJL8r1cbxdc6b/go-addr-util"
	yamux "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNWCEvi7bPRcvqAV8AKLGVNoQdArWi7NJayka2SM4XtRe/go-smux-yamux"
	discovery "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNh1kGFFdsPu79KNSaL4NUKUPb4Eiz4KHdMtFY6664RDp/go-libp2p/p2p/discovery"
	p2pbhost "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNh1kGFFdsPu79KNSaL4NUKUPb4Eiz4KHdMtFY6664RDp/go-libp2p/p2p/host/basic"
	rhost "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNh1kGFFdsPu79KNSaL4NUKUPb4Eiz4KHdMtFY6664RDp/go-libp2p/p2p/host/routed"
	identify "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNh1kGFFdsPu79KNSaL4NUKUPb4Eiz4KHdMtFY6664RDp/go-libp2p/p2p/protocol/identify"
	ping "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNh1kGFFdsPu79KNSaL4NUKUPb4Eiz4KHdMtFY6664RDp/go-libp2p/p2p/protocol/ping"
	u "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNiJuT8Ja3hMVpBHXv3Q6dwmperaQ6JjLtpMQgMCD7xvx/go-ipfs-util"
	p2phost "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNmJZL7FQySMtE2BQuLMuZg2EB2CLEunJJUSVSc9YnnbV/go-libp2p-host"
	logging "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRb5jh8z2E8hMGN2tkvs1yHynUanqnZ3UeKwgN1i9P1F8/go-log"
	goprocess "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	floodsub "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmSFihvoND3eDaAYRCeLgLPt62yCPgMZs1NSZmKFEtJQQw/go-libp2p-floodsub"
	mamask "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmSMZwvs3n4GBikZ7hKzT17c3bk65FmyZo2JqtJ16swqCv/multiaddr-filter"
	swarm "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmSwZMWwFZSUpe5muU2xgTUwppH24KfMwdPXiwbEp2c6G5/go-libp2p-swarm"
	routing "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTiWLZ6Fo5j4KcTVutZJ5KWRRJrbxzmxA4td8NfEdrPh7/go-libp2p-routing"
	circuit "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVTnHzuyECV9JzbXXfZRj1pKtgknp1esamUb2EH33mJkA/go-libp2p-circuit"
	mssmux "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVniQJkdzLZaZwzwMdd3dJTvWiJ1DQEkreVy6hs6h7Vk5/go-smux-multistream"
	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
	ds "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"
	pstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXauCuJzmzapetmC6W4TuDJLL1yFFrVzSHoWv8YdbmnxH/go-libp2p-peerstore"
	nilrouting "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXtoXbu9ReyV6Q4kDQ5CF9wXQNDY1PdHc4HhfxRR5AHB3/go-ipfs-routing/none"
	offroute "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXtoXbu9ReyV6Q4kDQ5CF9wXQNDY1PdHc4HhfxRR5AHB3/go-ipfs-routing/offline"
	dht "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmY1y2M1aCcVhy8UuTbZJBvuFbegZm47f9cDAdgxiehQfx/go-libp2p-kad-dht"
	smux "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmY9JXR3FupnYAYJWK9aMr9bCpqWKcToQ1tz8DVGTrHpHw/go-stream-muxer"
	connmgr "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZ1R2LxRZTUaeuMFEtQigzHfFCv3hLYBi5316aZ7YUeyf/go-libp2p-connmgr"
	ipnet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZPrWxuM8GHr4cGKbyF5CCT11sFUP9hgqpeUHALvx2nUr/go-libp2p-interface-pnet"
	peer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	bstore "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmaG4DZ4JaqEfvPWt5nPPgoTzhc1tr1T3f4Nu9Jpdm8ymY/go-ipfs-blockstore"
	ic "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	ifconnmgr "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmax8X1Kfahf5WfSB68EWDG3d3qyS3Sqs1v412fjPTfRwx/go-libp2p-interface-connmgr"
	mplex "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmc14vuKyGqX27RvBhekYytxSFJpaEgQVuVJgKSm69MEix/go-smux-multiplex"
	cid "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	exchange "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdcAXgEHUueP4A7b5hjabKn2EooeHgMreMvFC249dGCgc/go-ipfs-exchange-interface"
	metrics "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdeBtQGXjSt7cb97nx9JyLHHv5va2LyEAue7Q5tDFzpLy/go-libp2p-metrics"
	ipld "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
	pnet "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmenK8PgcpM2KYzEKnGx1LyN1QXawswM2F6HktCbWKuC1b/go-libp2p-pnet"
	mafilter "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmf2UAmRwDG4TvnkQpHZWPAzw7rpCYVhxmRXmYxXr5LD1g/go-maddr-filter"
)

const IpnsValidatorTag = "ipns"

const kReprovideFrequency = time.Hour * 12
const discoveryConnTimeout = time.Second * 30

var log = logging.Logger("core")

type mode int

const (
	// zero value is not a valid mode, must be explicitly set
	localMode mode = iota
	offlineMode
	onlineMode
)

func init() {
	identify.ClientVersion = "go-ipfs/" + config.CurrentVersionNumber + "/" + config.CurrentCommit
}

// IpfsNode is ONT-IPFS Core module. It represents an ONT-IPFS instance.
type IpfsNode struct {

	// Self
	Identity peer.ID // the local node's identity

	Repo repo.Repo

	// Local node
	Pinning         pin.Pinner // the pinning manager
	Mounts          Mounts     // current mount state, if any.
	PrivateKey      ic.PrivKey // the local node's private Key
	PNetFingerprint []byte     // fingerprint of private network

	// Services
	Peerstore  pstore.Peerstore     // storage for other Peer instances
	Blockstore bstore.GCBlockstore  // the block store (lower level)
	Filestore  *filestore.Filestore // the filestore blockstore
	BaseBlocks bstore.Blockstore    // the raw blockstore, no filestore wrapping
	GCLocker   bstore.GCLocker      // the locker used to protect the blockstore during gc
	Blocks     bserv.BlockService   // the block service, get/add blocks.
	DAG        ipld.DAGService      // the merkle dag service, get/add objects.
	Resolver   *resolver.Resolver   // the path resolution system
	Reporter   metrics.Reporter
	Discovery  discovery.Service
	FilesRoot  *mfs.Root

	// Online
	PeerHost     p2phost.Host        // the network host (server+client)
	Bootstrapper io.Closer           // the periodic bootstrapper
	Routing      routing.IpfsRouting // the routing system. recommend ipfs-dht
	Exchange     exchange.Interface  // the block exchange + strategy (bitswap)
	Namesys      namesys.NameSystem  // the name system, resolves paths to hashes
	Ping         *ping.PingService
	Reprovider   *rp.Reprovider // the value reprovider system
	IpnsRepub    *ipnsrp.Republisher

	Floodsub *floodsub.PubSub
	P2P      *p2p.P2P

	proc goprocess.Process
	ctx  context.Context

	mode         mode
	localModeSet bool
}

// Mounts defines what the node's mount state is. This should
// perhaps be moved to the daemon or mount. It's here because
// it needs to be accessible across daemon requests.
type Mounts struct {
	Ipfs mount.Mount
	Ipns mount.Mount
}

func (n *IpfsNode) startOnlineServices(ctx context.Context, routingOption RoutingOption, hostOption HostOption, do DiscoveryOption, pubsub, ipnsps, mplex bool) error {

	if n.PeerHost != nil { // already online.
		return errors.New("node already online")
	}

	// load private key
	if err := n.LoadPrivateKey(); err != nil {
		return err
	}

	// get undialable addrs from config
	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}
	var addrfilter []*net.IPNet
	for _, s := range cfg.Swarm.AddrFilters {
		f, err := mamask.NewMask(s)
		if err != nil {
			return fmt.Errorf("incorrectly formatted address filter in config: %s", s)
		}
		addrfilter = append(addrfilter, f)
	}

	if !cfg.Swarm.DisableBandwidthMetrics {
		// Set reporter
		n.Reporter = metrics.NewBandwidthCounter()
	}

	tpt := makeSmuxTransport(mplex)

	swarmkey, err := n.Repo.SwarmKey()
	if err != nil {
		return err
	}

	var protec ipnet.Protector
	if swarmkey != nil {
		protec, err = pnet.NewProtector(bytes.NewReader(swarmkey))
		if err != nil {
			return err
		}
		n.PNetFingerprint = protec.Fingerprint()
		go func() {
			t := time.NewTicker(30 * time.Second)
			<-t.C // swallow one tick
			for {
				select {
				case <-t.C:
					if ph := n.PeerHost; ph != nil {
						if len(ph.Network().Peers()) == 0 {
							log.Warning("We are in private network and have no peers.")
							log.Warning("This might be configuration mistake.")
						}
					}
				case <-n.Process().Closing():
					t.Stop()
					return
				}
			}
		}()
	}

	addrsFactory, err := makeAddrsFactory(cfg.Addresses)
	if err != nil {
		return err
	}

	connmgr, err := constructConnMgr(cfg.Swarm.ConnMgr)
	if err != nil {
		return err
	}

	hostopts := &ConstructPeerHostOpts{
		AddrsFactory:      addrsFactory,
		DisableNatPortMap: cfg.Swarm.DisableNatPortMap,
		DisableRelay:      cfg.Swarm.DisableRelay,
		EnableRelayHop:    cfg.Swarm.EnableRelayHop,
		ConnectionManager: connmgr,
	}
	peerhost, err := hostOption(ctx, n.Identity, n.Peerstore, n.Reporter,
		addrfilter, tpt, protec, hostopts)

	if err != nil {
		return err
	}

	if err := n.startOnlineServicesWithHost(ctx, peerhost, routingOption); err != nil {
		return err
	}

	// Ok, now we're ready to listen.
	if err := startListening(n.PeerHost, cfg); err != nil {
		return err
	}

	if pubsub || ipnsps {
		service, err := floodsub.NewFloodSub(ctx, peerhost)
		if err != nil {
			return err
		}
		n.Floodsub = service
	}

	if ipnsps {
		err = namesys.AddPubsubNameSystem(ctx, n.Namesys, n.PeerHost, n.Routing, n.Repo.Datastore(), n.Floodsub)
		if err != nil {
			return err
		}
	}

	n.P2P = p2p.NewP2P(n.Identity, n.PeerHost, n.Peerstore)

	// setup local discovery
	if do != nil {
		service, err := do(ctx, n.PeerHost)
		if err != nil {
			log.Error("mdns error: ", err)
		} else {
			service.RegisterNotifee(n)
			n.Discovery = service
		}
	}

	return n.Bootstrap(DefaultBootstrapConfig)
}

func constructConnMgr(cfg config.ConnMgr) (ifconnmgr.ConnManager, error) {
	switch cfg.Type {
	case "":
		// 'default' value is the basic connection manager
		return connmgr.NewConnManager(config.DefaultConnMgrLowWater, config.DefaultConnMgrHighWater, config.DefaultConnMgrGracePeriod), nil
	case "none":
		return nil, nil
	case "basic":
		grace, err := time.ParseDuration(cfg.GracePeriod)
		if err != nil {
			return nil, fmt.Errorf("parsing Swarm.ConnMgr.GracePeriod: %s", err)
		}

		return connmgr.NewConnManager(cfg.LowWater, cfg.HighWater, grace), nil
	default:
		return nil, fmt.Errorf("unrecognized ConnMgr.Type: %q", cfg.Type)
	}
}

func (n *IpfsNode) startLateOnlineServices(ctx context.Context) error {
	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}

	var keyProvider rp.KeyChanFunc

	switch cfg.Reprovider.Strategy {
	case "all":
		fallthrough
	case "":
		keyProvider = rp.NewBlockstoreProvider(n.Blockstore)
	case "roots":
		keyProvider = rp.NewPinnedProvider(n.Pinning, n.DAG, true)
	case "pinned":
		keyProvider = rp.NewPinnedProvider(n.Pinning, n.DAG, false)
	default:
		return fmt.Errorf("unknown reprovider strategy '%s'", cfg.Reprovider.Strategy)
	}
	n.Reprovider = rp.NewReprovider(ctx, n.Routing, keyProvider)

	reproviderInterval := kReprovideFrequency
	if cfg.Reprovider.Interval != "" {
		dur, err := time.ParseDuration(cfg.Reprovider.Interval)
		if err != nil {
			return err
		}

		reproviderInterval = dur
	}

	go n.Reprovider.Run(reproviderInterval)

	return nil
}

func makeAddrsFactory(cfg config.Addresses) (p2pbhost.AddrsFactory, error) {
	var annAddrs []ma.Multiaddr
	for _, addr := range cfg.Announce {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		annAddrs = append(annAddrs, maddr)
	}

	filters := mafilter.NewFilters()
	noAnnAddrs := map[string]bool{}
	for _, addr := range cfg.NoAnnounce {
		f, err := mamask.NewMask(addr)
		if err == nil {
			filters.AddDialFilter(f)
			continue
		}
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		noAnnAddrs[maddr.String()] = true
	}

	return func(allAddrs []ma.Multiaddr) []ma.Multiaddr {
		var addrs []ma.Multiaddr
		if len(annAddrs) > 0 {
			addrs = annAddrs
		} else {
			addrs = allAddrs
		}

		var out []ma.Multiaddr
		for _, maddr := range addrs {
			// check for exact matches
			ok, _ := noAnnAddrs[maddr.String()]
			// check for /ipcidr matches
			if !ok && !filters.AddrBlocked(maddr) {
				out = append(out, maddr)
			}
		}
		return out
	}, nil
}

func makeSmuxTransport(mplexExp bool) smux.Transport {
	mstpt := mssmux.NewBlankTransport()

	ymxtpt := &yamux.Transport{
		AcceptBacklog:          512,
		ConnectionWriteTimeout: time.Second * 10,
		KeepAliveInterval:      time.Second * 30,
		EnableKeepAlive:        true,
		MaxStreamWindowSize:    uint32(1024 * 512),
		LogOutput:              ioutil.Discard,
	}

	if os.Getenv("YAMUX_DEBUG") != "" {
		ymxtpt.LogOutput = os.Stderr
	}

	mstpt.AddTransport("/yamux/1.0.0", ymxtpt)

	if mplexExp {
		mstpt.AddTransport("/mplex/6.7.0", mplex.DefaultTransport)
	}

	// Allow muxer preference order overriding
	if prefs := os.Getenv("LIBP2P_MUX_PREFS"); prefs != "" {
		mstpt.OrderPreference = strings.Fields(prefs)
	}

	return mstpt
}

func setupDiscoveryOption(d config.Discovery) DiscoveryOption {
	if d.MDNS.Enabled {
		return func(ctx context.Context, h p2phost.Host) (discovery.Service, error) {
			if d.MDNS.Interval == 0 {
				d.MDNS.Interval = 5
			}
			return discovery.NewMdnsService(ctx, h, time.Duration(d.MDNS.Interval)*time.Second, discovery.ServiceTag)
		}
	}
	return nil
}

// HandlePeerFound attempts to connect to peer from `PeerInfo`, if it fails
// logs a warning log.
func (n *IpfsNode) HandlePeerFound(p pstore.PeerInfo) {
	log.Warning("trying peer info: ", p)
	ctx, cancel := context.WithTimeout(n.Context(), discoveryConnTimeout)
	defer cancel()
	if err := n.PeerHost.Connect(ctx, p); err != nil {
		log.Warning("Failed to connect to peer found by discovery: ", err)
	}
}

// startOnlineServicesWithHost  is the set of services which need to be
// initialized with the host and _before_ we start listening.
func (n *IpfsNode) startOnlineServicesWithHost(ctx context.Context, host p2phost.Host, routingOption RoutingOption) error {
	// setup diagnostics service
	n.Ping = ping.NewPingService(host)

	// setup routing service
	r, err := routingOption(ctx, host, n.Repo.Datastore())
	if err != nil {
		return err
	}
	n.Routing = r

	// Wrap standard peer host with routing system to allow unknown peer lookups
	n.PeerHost = rhost.Wrap(host, n.Routing)

	// setup exchange service
	bitswapNetwork := bsnet.NewFromIpfsHost(n.PeerHost, n.Routing)
	n.Exchange = bitswap.New(ctx, bitswapNetwork, n.Blockstore)

	size, err := n.getCacheSize()
	if err != nil {
		return err
	}

	// setup name system
	n.Namesys = namesys.NewNameSystem(n.Routing, n.Repo.Datastore(), size)

	// setup ipns republishing
	return n.setupIpnsRepublisher()
}

// getCacheSize returns cache life and cache size
func (n *IpfsNode) getCacheSize() (int, error) {
	cfg, err := n.Repo.Config()
	if err != nil {
		return 0, err
	}

	cs := cfg.Ipns.ResolveCacheSize
	if cs == 0 {
		cs = 128
	}
	if cs < 0 {
		return 0, fmt.Errorf("cannot specify negative resolve cache size")
	}
	return cs, nil
}

func (n *IpfsNode) setupIpnsRepublisher() error {
	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}

	n.IpnsRepub = ipnsrp.NewRepublisher(n.Routing, n.Repo.Datastore(), n.PrivateKey, n.Repo.Keystore())

	if cfg.Ipns.RepublishPeriod != "" {
		d, err := time.ParseDuration(cfg.Ipns.RepublishPeriod)
		if err != nil {
			return fmt.Errorf("failure to parse config setting IPNS.RepublishPeriod: %s", err)
		}

		if !u.Debug && (d < time.Minute || d > (time.Hour*24)) {
			return fmt.Errorf("config setting IPNS.RepublishPeriod is not between 1min and 1day: %s", d)
		}

		n.IpnsRepub.Interval = d
	}

	if cfg.Ipns.RecordLifetime != "" {
		d, err := time.ParseDuration(cfg.Ipns.RepublishPeriod)
		if err != nil {
			return fmt.Errorf("failure to parse config setting IPNS.RecordLifetime: %s", err)
		}

		n.IpnsRepub.RecordLifetime = d
	}

	n.Process().Go(n.IpnsRepub.Run)

	return nil
}

// Process returns the Process object
func (n *IpfsNode) Process() goprocess.Process {
	return n.proc
}

// Close calls Close() on the Process object
func (n *IpfsNode) Close() error {
	return n.proc.Close()
}

// Context returns the IpfsNode context
func (n *IpfsNode) Context() context.Context {
	if n.ctx == nil {
		n.ctx = context.TODO()
	}
	return n.ctx
}

// teardown closes owned children. If any errors occur, this function returns
// the first error.
func (n *IpfsNode) teardown() error {
	log.Debug("core is shutting down...")
	// owned objects are closed in this teardown to ensure that they're closed
	// regardless of which constructor was used to add them to the node.
	var closers []io.Closer

	// NOTE: The order that objects are added(closed) matters, if an object
	// needs to use another during its shutdown/cleanup process, it should be
	// closed before that other object

	if n.FilesRoot != nil {
		closers = append(closers, n.FilesRoot)
	}

	if n.Exchange != nil {
		closers = append(closers, n.Exchange)
	}

	if n.Mounts.Ipfs != nil && !n.Mounts.Ipfs.IsActive() {
		closers = append(closers, mount.Closer(n.Mounts.Ipfs))
	}
	if n.Mounts.Ipns != nil && !n.Mounts.Ipns.IsActive() {
		closers = append(closers, mount.Closer(n.Mounts.Ipns))
	}

	if dht, ok := n.Routing.(*dht.IpfsDHT); ok {
		closers = append(closers, dht.Process())
	}

	if n.Blocks != nil {
		closers = append(closers, n.Blocks)
	}

	if n.Bootstrapper != nil {
		closers = append(closers, n.Bootstrapper)
	}

	if n.PeerHost != nil {
		closers = append(closers, n.PeerHost)
	}

	// Repo closed last, most things need to preserve state here
	closers = append(closers, n.Repo)

	var errs []error
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// OnlineMode returns whether or not the IpfsNode is in OnlineMode.
func (n *IpfsNode) OnlineMode() bool {
	switch n.mode {
	case onlineMode:
		return true
	default:
		return false
	}
}

// SetLocal will set the IpfsNode to local mode
func (n *IpfsNode) SetLocal(isLocal bool) {
	if isLocal {
		n.mode = localMode
	}
	n.localModeSet = true
}

// LocalMode returns whether or not the IpfsNode is in LocalMode
func (n *IpfsNode) LocalMode() bool {
	if !n.localModeSet {
		// programmer error should not happen
		panic("local mode not set")
	}
	switch n.mode {
	case localMode:
		return true
	default:
		return false
	}
}

// Bootstrap will set and call the IpfsNodes bootstrap function.
func (n *IpfsNode) Bootstrap(cfg BootstrapConfig) error {

	// TODO what should return value be when in offlineMode?
	if n.Routing == nil {
		return nil
	}

	if n.Bootstrapper != nil {
		n.Bootstrapper.Close() // stop previous bootstrap process.
	}

	// if the caller did not specify a bootstrap peer function, get the
	// freshest bootstrap peers from config. this responds to live changes.
	if cfg.BootstrapPeers == nil {
		cfg.BootstrapPeers = func() []pstore.PeerInfo {
			ps, err := n.loadBootstrapPeers()
			if err != nil {
				log.Warning("failed to parse bootstrap peers from config")
				return nil
			}
			return ps
		}
	}

	var err error
	n.Bootstrapper, err = Bootstrap(n, cfg)
	return err
}

func (n *IpfsNode) loadID() error {
	if n.Identity != "" {
		return errors.New("identity already loaded")
	}

	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}

	cid := cfg.Identity.PeerID
	if cid == "" {
		return errors.New("identity was not set in config (was 'ipfs init' run?)")
	}
	if len(cid) == 0 {
		return errors.New("no peer ID in config! (was 'ipfs init' run?)")
	}

	id, err := peer.IDB58Decode(cid)
	if err != nil {
		return fmt.Errorf("peer ID invalid: %s", err)
	}

	n.Identity = id
	return nil
}

// GetKey will return a key from the Keystore with name `name`.
func (n *IpfsNode) GetKey(name string) (ic.PrivKey, error) {
	if name == "self" {
		return n.PrivateKey, nil
	} else {
		return n.Repo.Keystore().Get(name)
	}
}

func (n *IpfsNode) LoadPrivateKey() error {
	if n.Identity == "" || n.Peerstore == nil {
		return errors.New("loaded private key out of order")
	}

	if n.PrivateKey != nil {
		return errors.New("private key already loaded")
	}

	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}

	sk, err := loadPrivateKey(&cfg.Identity, n.Identity)
	if err != nil {
		return err
	}

	n.PrivateKey = sk
	n.Peerstore.AddPrivKey(n.Identity, n.PrivateKey)
	n.Peerstore.AddPubKey(n.Identity, sk.GetPublic())
	return nil
}

func (n *IpfsNode) loadBootstrapPeers() ([]pstore.PeerInfo, error) {
	cfg, err := n.Repo.Config()
	if err != nil {
		return nil, err
	}

	parsed, err := cfg.BootstrapPeers()
	if err != nil {
		return nil, err
	}
	return toPeerInfos(parsed), nil
}

func (n *IpfsNode) loadFilesRoot() error {
	dsk := ds.NewKey("/local/filesroot")
	pf := func(ctx context.Context, c *cid.Cid) error {
		return n.Repo.Datastore().Put(dsk, c.Bytes())
	}

	var nd *merkledag.ProtoNode
	val, err := n.Repo.Datastore().Get(dsk)

	switch {
	case err == ds.ErrNotFound || val == nil:
		nd = ft.EmptyDirNode()
		err := n.DAG.Add(n.Context(), nd)
		if err != nil {
			return fmt.Errorf("failure writing to dagstore: %s", err)
		}
	case err == nil:
		c, err := cid.Cast(val.([]byte))
		if err != nil {
			return err
		}

		rnd, err := n.DAG.Get(n.Context(), c)
		if err != nil {
			return fmt.Errorf("error loading filesroot from DAG: %s", err)
		}

		pbnd, ok := rnd.(*merkledag.ProtoNode)
		if !ok {
			return merkledag.ErrNotProtobuf
		}

		nd = pbnd
	default:
		return err
	}

	mr, err := mfs.NewRoot(n.Context(), n.DAG, nd, pf)
	if err != nil {
		return err
	}

	n.FilesRoot = mr
	return nil
}

// SetupOfflineRouting loads the local nodes private key and
// uses it to instantiate a routing system in offline mode.
// This is primarily used for offline ipns modifications.
func (n *IpfsNode) SetupOfflineRouting() error {
	if n.Routing != nil {
		// Routing was already set up
		return nil
	}
	err := n.LoadPrivateKey()
	if err != nil {
		return err
	}

	n.Routing = offroute.NewOfflineRouter(n.Repo.Datastore(), n.PrivateKey)

	size, err := n.getCacheSize()
	if err != nil {
		return err
	}

	n.Namesys = namesys.NewNameSystem(n.Routing, n.Repo.Datastore(), size)

	return nil
}

func loadPrivateKey(cfg *config.Identity, id peer.ID) (ic.PrivKey, error) {
	sk, err := cfg.DecodePrivateKey("passphrase todo!")
	if err != nil {
		return nil, err
	}

	id2, err := peer.IDFromPrivateKey(sk)
	if err != nil {
		return nil, err
	}

	if id2 != id {
		return nil, fmt.Errorf("private key in config does not match id: %s != %s", id, id2)
	}

	return sk, nil
}

func listenAddresses(cfg *config.Config) ([]ma.Multiaddr, error) {
	var listen []ma.Multiaddr
	for _, addr := range cfg.Addresses.Swarm {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("failure to parse config.Addresses.Swarm: %s", cfg.Addresses.Swarm)
		}
		listen = append(listen, maddr)
	}

	return listen, nil
}

type ConstructPeerHostOpts struct {
	AddrsFactory      p2pbhost.AddrsFactory
	DisableNatPortMap bool
	DisableRelay      bool
	EnableRelayHop    bool
	ConnectionManager ifconnmgr.ConnManager
}

type HostOption func(ctx context.Context, id peer.ID, ps pstore.Peerstore, bwr metrics.Reporter, fs []*net.IPNet, tpt smux.Transport, protc ipnet.Protector, opts *ConstructPeerHostOpts) (p2phost.Host, error)

var DefaultHostOption HostOption = constructPeerHost

// isolates the complex initialization steps
func constructPeerHost(ctx context.Context, id peer.ID, ps pstore.Peerstore, bwr metrics.Reporter, fs []*net.IPNet, tpt smux.Transport, protec ipnet.Protector, opts *ConstructPeerHostOpts) (p2phost.Host, error) {

	// no addresses to begin with. we'll start later.
	swrm, err := swarm.NewSwarmWithProtector(ctx, nil, id, ps, protec, tpt, bwr)
	if err != nil {
		return nil, err
	}

	network := (*swarm.Network)(swrm)

	for _, f := range fs {
		network.Swarm().Filters.AddDialFilter(f)
	}

	hostOpts := []interface{}{bwr}
	if !opts.DisableNatPortMap {
		hostOpts = append(hostOpts, p2pbhost.NATPortMap)
	}
	if opts.ConnectionManager != nil {
		hostOpts = append(hostOpts, opts.ConnectionManager)
	}

	addrsFactory := opts.AddrsFactory
	if !opts.DisableRelay {
		if addrsFactory != nil {
			addrsFactory = composeAddrsFactory(addrsFactory, filterRelayAddrs)
		} else {
			addrsFactory = filterRelayAddrs
		}
	}

	if addrsFactory != nil {
		hostOpts = append(hostOpts, addrsFactory)
	}

	host := p2pbhost.New(network, hostOpts...)

	if !opts.DisableRelay {
		var relayOpts []circuit.RelayOpt
		if opts.EnableRelayHop {
			relayOpts = append(relayOpts, circuit.OptHop)
		}

		err := circuit.AddRelayTransport(ctx, host, relayOpts...)
		if err != nil {
			host.Close()
			return nil, err
		}
	}

	return host, nil
}

func filterRelayAddrs(addrs []ma.Multiaddr) []ma.Multiaddr {
	var raddrs []ma.Multiaddr
	for _, addr := range addrs {
		_, err := addr.ValueForProtocol(circuit.P_CIRCUIT)
		if err == nil {
			continue
		}
		raddrs = append(raddrs, addr)
	}
	return raddrs
}

func composeAddrsFactory(f, g p2pbhost.AddrsFactory) p2pbhost.AddrsFactory {
	return func(addrs []ma.Multiaddr) []ma.Multiaddr {
		return f(g(addrs))
	}
}

// startListening on the network addresses
func startListening(host p2phost.Host, cfg *config.Config) error {
	listenAddrs, err := listenAddresses(cfg)
	if err != nil {
		return err
	}

	// make sure we error out if our config does not have addresses we can use
	log.Debugf("Config.Addresses.Swarm:%s", listenAddrs)
	filteredAddrs := addrutil.FilterUsableAddrs(listenAddrs)
	log.Debugf("Config.Addresses.Swarm:%s (filtered)", filteredAddrs)
	if len(filteredAddrs) < 1 {
		return fmt.Errorf("addresses in config not usable: %s", listenAddrs)
	}

	// Actually start listening:
	if err := host.Network().Listen(filteredAddrs...); err != nil {
		return err
	}

	// list out our addresses
	addrs, err := host.Network().InterfaceListenAddresses()
	if err != nil {
		return err
	}
	log.Infof("Swarm listening at: %s", addrs)
	return nil
}

func constructDHTRouting(ctx context.Context, host p2phost.Host, dstore ds.Batching) (routing.IpfsRouting, error) {
	dhtRouting := dht.NewDHT(ctx, host, dstore)
	dhtRouting.Validator[IpnsValidatorTag] = namesys.NewIpnsRecordValidator(host.Peerstore())
	dhtRouting.Selector[IpnsValidatorTag] = namesys.IpnsSelectorFunc
	return dhtRouting, nil
}

func constructClientDHTRouting(ctx context.Context, host p2phost.Host, dstore ds.Batching) (routing.IpfsRouting, error) {
	dhtRouting := dht.NewDHTClient(ctx, host, dstore)
	dhtRouting.Validator[IpnsValidatorTag] = namesys.NewIpnsRecordValidator(host.Peerstore())
	dhtRouting.Selector[IpnsValidatorTag] = namesys.IpnsSelectorFunc
	return dhtRouting, nil
}

type RoutingOption func(context.Context, p2phost.Host, ds.Batching) (routing.IpfsRouting, error)

type DiscoveryOption func(context.Context, p2phost.Host) (discovery.Service, error)

var DHTOption RoutingOption = constructDHTRouting
var DHTClientOption RoutingOption = constructClientDHTRouting
var NilRouterOption RoutingOption = nilrouting.ConstructNilRouting
