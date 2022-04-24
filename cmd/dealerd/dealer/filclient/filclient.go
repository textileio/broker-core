package filclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/libp2p/go-libp2p-core/host"
	inet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	record "github.com/libp2p/go-libp2p-record"
	routed "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/filapi"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/metric"
)

var (
	log         = logger.Logger("dealer/filclient")
	boostrpeers = []string{
		"/dns4/bootstrap-0.mainnet.filops.net/tcp/1347/p2p/12D3KooWCVe8MmsEMes2FzgTpt9fXtmCY7wrq91GRiaC8PHSCCBj",
		"/dns4/bootstrap-1.mainnet.filops.net/tcp/1347/p2p/12D3KooWCwevHg1yLCvktf2nvLu7L9894mcrJR4MsBCcm4syShVc",
		"/dns4/bootstrap-2.mainnet.filops.net/tcp/1347/p2p/12D3KooWEWVwHGn2yR36gKLozmb4YjDJGerotAPGxmdWZx2nxMC4",
		"/dns4/bootstrap-3.mainnet.filops.net/tcp/1347/p2p/12D3KooWKhgq8c7NQ9iGjbyK7v7phXvG6492HQfiDaGHLHLQjk7R",
		"/dns4/bootstrap-4.mainnet.filops.net/tcp/1347/p2p/12D3KooWL6PsFNPhYftrJzGgF5U18hFoaVhfGk7xwzD8yVrHJ3Uc",
		"/dns4/bootstrap-5.mainnet.filops.net/tcp/1347/p2p/12D3KooWLFynvDQiUpXoHroV1YxKHhPJgysQGH2k3ZGwtWzR4dFH",
		"/dns4/bootstrap-6.mainnet.filops.net/tcp/1347/p2p/12D3KooWP5MwCiqdMETF9ub1P3MbCvQCcfconnYHbWg6sUJcDRQQ",
		"/dns4/bootstrap-7.mainnet.filops.net/tcp/1347/p2p/12D3KooWRs3aY1p3juFjPy8gPN95PEQChm2QKGUCAdcDCC4EBMKf",
		"/dns4/bootstrap-8.mainnet.filops.net/tcp/1347/p2p/12D3KooWScFR7385LTyR4zU1bYdzSiiAb5rnNABfVahPvVSzyTkR",

		"/dns4/lotus-bootstrap.ipfsforce.com/tcp/41778/p2p/12D3KooWGhufNmZHF3sv48aQeS13ng5XVJZ9E6qy2Ms4VzqeUsHk",
		"/dns4/bootstrap-0.starpool.in/tcp/12757/p2p/12D3KooWDqaZkm3oSczUm3dvAJ5aL2rdSeQ5VQbnHRTQNEFShhmc",
		"/dns4/bootstrap-1.starpool.in/tcp/12757/p2p/12D3KooWDgQrcyZpcMAkbEFSJJYV2qXEMwXX67WTbqpNdbifHaEq",
		"/dns4/node.glif.io/tcp/1235/p2p/12D3KooWBF8cpp65hp2u9LK5mh19x67ftAam84z9LsfaquTDSBpt",
		"/dns4/bootstrap-0.ipfsmain.cn/tcp/34721/p2p/12D3KooWQnwEGNqcM2nAcPtRR9rAX8Hrg4k9kJLCHoTR5chJfz6d",
		"/dns4/bootstrap-1.ipfsmain.cn/tcp/34723/p2p/12D3KooWMKxMkD5DMpSWsW7dBddKxKT7L2GgbNuckz9otxvkvByP",
	}
	bootstrapPeers []peer.AddrInfo
)

func init() {
	for _, bs := range boostrpeers {
		ma, err := multiaddr.NewMultiaddr(bs)
		if err != nil {
			log.Errorf("parsing bootstrap address: ", err)
			continue
		}
		ai, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			log.Errorf("create address info: ", err)
			continue
		}
		bootstrapPeers = append(bootstrapPeers, *ai)
	}
}

// FilClient provides API to interact with the Filecoin network.
type FilClient struct {
	conf config

	api  filapi.FilAPI
	host host.Host

	metricExecAuctionDeal                    metric.Int64Counter
	metricGetChainHeight                     metric.Int64Counter
	metricResolveDealIDFromMessage           metric.Int64Counter
	metricCheckDealStatusWithStorageProvider metric.Int64Counter
	metricCheckChainDeal                     metric.Int64Counter

	metricFilAPIRequests       metric.Int64Counter
	metricFilAPIDurationMillis metric.Int64Histogram
}

// New returns a new FilClient.
func New(api v0api.FullNode, h host.Host, opts ...Option) (*FilClient, error) {
	cfg := defaultConfig
	for _, op := range opts {
		if err := op(&cfg); err != nil {
			return nil, fmt.Errorf("applying option: %s", err)
		}
	}

	filopts := []dht.Option{dht.Mode(dht.ModeAuto),
		dht.Validator(record.NamespacedValidator{
			"pk": record.PublicKeyValidator{},
		}),
		dht.ProtocolPrefix("/fil/kad/testnetnet"),
		dht.QueryFilter(dht.PublicQueryFilter),
		dht.RoutingTableFilter(dht.PublicRoutingTableFilter),
		dht.DisableProviders(),
		dht.BootstrapPeers(bootstrapPeers...),
		dht.DisableValues()}
	fildht, err := dht.New(context.Background(), h, filopts...)
	if err != nil {
		return nil, fmt.Errorf("creating dht client: %s", err)
	}
	h = routed.Wrap(h, fildht)

	var wg sync.WaitGroup
	wg.Add(len(bootstrapPeers))
	for _, bp := range bootstrapPeers {
		bp := bp
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			if err := h.Connect(ctx, bp); err != nil {
				log.Errorf("bootstrap peer connect: %s", err)
			}
		}()
	}
	wg.Wait()

	fc := &FilClient{
		conf: cfg,
		host: h,
	}
	fc.api = filapi.Measured(api, fc.collectAPIMetrics)
	fc.initMetrics()

	return fc, nil
}

func (fc *FilClient) streamToStorageProvider(
	ctx context.Context,
	maddr address.Address,
	protocol protocol.ID) (inet.Stream, error) {
	mpid, err := fc.connectToStorageProvider(ctx, maddr)
	if err != nil {
		return nil, fmt.Errorf("connecting with storage-provider: %s", err)
	}

	s, err := fc.host.NewStream(ctx, mpid, protocol)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream to peer: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		if err := s.SetDeadline(deadline); err != nil {
			return nil, fmt.Errorf("set deadline of stream: %s", err)
		}
	}

	return s, nil
}

func (fc *FilClient) connectToStorageProvider(ctx context.Context, maddr address.Address) (peer.ID, error) {
	minfo, err := fc.api.StateMinerInfo(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return "", fmt.Errorf("state storage-provider info call: %s", err)
	}

	if minfo.PeerId == nil {
		log.Warnf("storage-provider %s doesn't have a PeerID on-chain", maddr)
		return "", fmt.Errorf("storage-provider %s has no peer ID set", maddr)
	}

	addrInfo, err := fc.api.NetFindPeer(ctx, *minfo.PeerId)
	if err != nil {
		log.Warnf("net-find-peer %s api call failed: %s", *minfo.PeerId, err)
	}

	if err := fc.host.Connect(ctx, peer.AddrInfo{
		ID:    *minfo.PeerId,
		Addrs: addrInfo.Addrs,
	}); err != nil {
		log.Warnf("failed connecting with storage-provider %s", maddr)
		return "", fmt.Errorf("connecting to storage-provider: %s", err)
	}

	return *minfo.PeerId, nil
}

func (fc *FilClient) connectToRemoteWallet(ctx context.Context, rw *store.RemoteWallet) (peer.ID, error) {
	peerID, err := peer.Decode(rw.PeerID)
	if err != nil {
		return "", fmt.Errorf("decoding remote peer-id: %s", err)
	}
	maddrs := make([]multiaddr.Multiaddr, len(rw.Multiaddrs))
	for i, maddr := range rw.Multiaddrs {
		maddr, err := multiaddr.NewMultiaddr(maddr)
		if err != nil {
			log.Warnf("parsing multiaddr %s: %s", maddr, err)
			continue
		}
		maddrs[i] = maddr
	}

	if fc.conf.relayMaddr != "" {
		relayed, err := multiaddr.NewMultiaddr(fc.conf.relayMaddr + "/p2p-circuit/p2p/" + peerID.String())
		if err != nil {
			return "", fmt.Errorf("creating relayed maddr: %s", err)
		}
		maddrs = append(maddrs, relayed)
	}
	pi := peer.AddrInfo{
		ID:    peerID,
		Addrs: maddrs,
	}
	if err := fc.host.Connect(ctx, pi); err != nil {
		return "", fmt.Errorf("connecting with remote wallet: %s", err)
	}

	return peerID, nil
}
