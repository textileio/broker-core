package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	cconnmgr "github.com/libp2p/go-libp2p-core/connmgr"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	ps "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/market"
	"github.com/textileio/broker-core/pubsub"
	badger "github.com/textileio/go-ds-badger3"
)

type config struct {
	HostAddr    multiaddr.Multiaddr
	ConnManager cconnmgr.ConnManager
	RepoPath    string
}

type Option func(c *config) error

func WithHostAddr(addr multiaddr.Multiaddr) Option {
	return func(c *config) error {
		c.HostAddr = addr
		return nil
	}
}

func WithRepoPath(path string) Option {
	return func(c *config) error {
		c.RepoPath = path
		return nil
	}
}

func WithConnectionManager(manager cconnmgr.ConnManager) Option {
	return func(c *config) error {
		c.ConnManager = manager
		return nil
	}
}

func setDefaults(conf *config) error {
	if conf.HostAddr == nil {
		addr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
		if err != nil {
			return err
		}
		conf.HostAddr = addr
	}
	if conf.ConnManager == nil {
		conf.ConnManager = connmgr.NewConnManager(100, 400, time.Second*20)
	}
	return nil
}

type Service struct {
	peer      *ipfslite.Peer
	deals     *pubsub.Topic
	finalizer *finalizer.Finalizer
}

func New(opts ...Option) (*Service, error) {
	var (
		conf config
		fin  = finalizer.NewFinalizer()
	)

	for _, opt := range opts {
		if err := opt(&conf); err != nil {
			return nil, err
		}
	}
	if err := setDefaults(&conf); err != nil {
		return nil, fmt.Errorf("setting defaults: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	fin.Add(finalizer.NewContextCloser(cancel))

	// Setup ipfslite peerstore
	lstore, err := badgerStore(filepath.Join(conf.RepoPath, "ipfslite"), fin)
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
	hostKey, err := getHostKey(conf)
	if err != nil {
		return nil, fin.Cleanupf("getting host key: %v", err)
	}
	host, dht, err := ipfslite.SetupLibp2p(
		ctx,
		hostKey,
		nil,
		[]multiaddr.Multiaddr{conf.HostAddr},
		lstore,
		libp2p.Peerstore(pstore),
		libp2p.ConnectionManager(conf.ConnManager),
		libp2p.DisableRelay(),
	)
	if err != nil {
		return nil, fin.Cleanupf("setting up libp2p", err)
	}

	// Create ipfslite peer
	lpeer, err := ipfslite.New(ctx, lstore, host, dht, nil)
	if err != nil {
		return nil, fin.Cleanupf("creating ipfslite peer", err)
	}

	// Setup pubsub
	gps, err := ps.NewGossipSub(ctx, host)
	if err != nil {
		return nil, fin.Cleanupf("starting libp2p pubsub: %v", err)
	}

	// Subscribe to market deals
	deals, err := pubsub.NewTopic(ctx, gps, host.ID(), string(market.ProtocolDeals), true)
	if err != nil {
		cancel()
		return nil, fin.Cleanupf("creating deals topic: %v", err)
	}

	s := &Service{
		peer:      lpeer,
		deals:     deals,
		finalizer: fin,
	}

	// deals.SetEventHandler(s.eventHandler)
	// deals.SetMessageHandler(s.messageHandler)

	return s, nil
}

func (s *Service) Bootstrap(addrs []peer.AddrInfo) {
	s.peer.Bootstrap(addrs)
}

func (s *Service) GetPeer() *ipfslite.Peer {
	return s.peer
}

func (s *Service) Close() error {
	err := s.deals.Close()
	return s.finalizer.Cleanup(err)
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

func getHostKey(conf config) (crypto.PrivKey, error) {
	dir := filepath.Join(conf.RepoPath, "ipfslite")
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

func badgerStore(repoPath string, fin *finalizer.Finalizer) (datastore.Batching, error) {
	if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
		return nil, err
	}
	dstore, err := badger.NewDatastore(repoPath, &badger.DefaultOptions)
	if err != nil {
		return nil, err
	}
	fin.Add(dstore)
	return dstore, nil
}
