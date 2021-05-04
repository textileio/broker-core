package marketpeer

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	ipfslite "github.com/hsanjuan/ipfs-lite"
	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	cconnmgr "github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	ps "github.com/libp2p/go-libp2p-pubsub"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/marketpeer/mdns"
	"github.com/textileio/broker-core/pubsub"
	badger "github.com/textileio/go-ds-badger3"
)

var log = golog.Logger("mpeer")

// Config defines params for Peer configuration.
type Config struct {
	RepoPath      string
	HostMultiaddr string
	ConnManager   cconnmgr.ConnManager
}

func setDefaults(conf *Config) {
	if len(conf.HostMultiaddr) == 0 {
		conf.HostMultiaddr = "/ip4/0.0.0.0/tcp/0"
	}
	if conf.ConnManager == nil {
		conf.ConnManager = connmgr.NewConnManager(100, 400, time.Second*20)
	}
}

// Peer wraps libp2p peer components needed to partake in the broker market.
type Peer struct {
	host      host.Host
	peer      *ipfslite.Peer
	ps        *ps.PubSub
	finalizer *finalizer.Finalizer
}

// New returns a new Peer.
func New(conf Config) (*Peer, error) {
	setDefaults(&conf)

	hostAddr, err := multiaddr.NewMultiaddr(conf.HostMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("parsing host multiaddress: %s", err)
	}

	fin := finalizer.NewFinalizer()
	ctx, cancel := context.WithCancel(context.Background())
	fin.Add(finalizer.NewContextCloser(cancel))

	// Setup ipfslite peerstore
	repoPath := filepath.Join(conf.RepoPath, "ipfslite")
	if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("making dir: %v", err)
	}
	lstore, err := badger.NewDatastore(repoPath, &badger.DefaultOptions)
	if err != nil {
		return nil, fin.Cleanupf("creating repo: %v", err)
	}
	fin.Add(lstore)
	pstore, err := pstoreds.NewPeerstore(ctx, lstore, pstoreds.DefaultOpts())
	if err != nil {
		return nil, fin.Cleanupf("creating peerstore: %v", err)
	}
	fin.Add(pstore)

	// Setup libp2p
	hostKey, err := getHostKey(conf.RepoPath)
	if err != nil {
		return nil, fin.Cleanupf("getting host key: %v", err)
	}
	lhost, dht, err := ipfslite.SetupLibp2p(
		ctx,
		hostKey,
		nil,
		[]multiaddr.Multiaddr{hostAddr},
		lstore,
		libp2p.Peerstore(pstore),
		libp2p.ConnectionManager(conf.ConnManager),
		libp2p.DefaultTransports,
		libp2p.Transport(quic.NewTransport),
		libp2p.DisableRelay(),
	)
	if err != nil {
		return nil, fin.Cleanupf("setting up libp2p", err)
	}
	fin.Add(lhost, dht)

	// Create ipfslite peer
	lpeer, err := ipfslite.New(ctx, lstore, lhost, dht, nil)
	if err != nil {
		return nil, fin.Cleanupf("creating ipfslite peer", err)
	}

	// Setup pubsub
	gps, err := ps.NewGossipSub(ctx, lhost)
	if err != nil {
		return nil, fin.Cleanupf("starting libp2p pubsub: %v", err)
	}

	log.Infof("marketpeer %s is online", lhost.ID())

	return &Peer{
		host:      lhost,
		peer:      lpeer,
		ps:        gps,
		finalizer: fin,
	}, nil
}

// Close the peer.
func (p *Peer) Close() error {
	return p.finalizer.Cleanup(nil)
}

// Self returns the peer's id.
func (p *Peer) Self() peer.ID {
	return p.host.ID()
}

// Bootstrap the market peer against well-known network peers.
func (p *Peer) Bootstrap() {
	p.peer.Bootstrap(ipfslite.DefaultBootstrapPeers())
	log.Info("peer was bootstapped")
}

// EnableMDNS enables an MDNS discovery service.
// This is useful on a local network (testing).
func (p *Peer) EnableMDNS(internalSecs int) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.finalizer.Add(finalizer.NewContextCloser(cancel))

	if err := mdns.Start(ctx, p.host, internalSecs); err != nil {
		return err
	}

	log.Infof("mdns was enabled (interval=%ds)", internalSecs)
	return nil
}

// NewTopic returns a new pubsub.Topic using the peer's host.
func (p *Peer) NewTopic(ctx context.Context, topic string, subscribe bool) (*pubsub.Topic, error) {
	return pubsub.NewTopic(ctx, p.ps, p.host.ID(), topic, subscribe)
}

func newHostKey() (crypto.PrivKey, []byte, error) {
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	key, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}
	return priv, key, nil
}

func getHostKey(repoPath string) (crypto.PrivKey, error) {
	dir := filepath.Join(repoPath, "ipfslite")
	pth := filepath.Join(dir, "key")
	_, err := os.Stat(pth)
	if os.IsNotExist(err) {
		key, bytes, err := newHostKey()
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, err
		}
		if err = ioutil.WriteFile(pth, bytes, 0400); err != nil {
			return nil, err
		}
		return key, nil
	} else if err != nil {
		return nil, err
	} else {
		bytes, err := ioutil.ReadFile(pth)
		if err != nil {
			return nil, err
		}
		return crypto.UnmarshalPrivateKey(bytes)
	}
}
