package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/cmd/dealerd/dealer"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/filclient"
	"github.com/textileio/broker-core/cmd/dealerd/dealermock"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	dealeri "github.com/textileio/broker-core/dealer"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/go-libp2p-pubsub-rpc/finalizer"
	golog "github.com/textileio/go-log/v2"
)

var log = golog.Logger("dealer/service")

// Config defines params for Service configuration.
type Config struct {
	PostgresURI string

	LotusGatewayURL         []string
	LotusExportedWalletAddr string

	AllowUnverifiedDeals             bool
	MaxVerifiedPricePerGiBPerEpoch   int64
	MaxUnverifiedPricePerGiBPerEpoch int64
	RelayMaddr                       string

	Mock bool
}

// Service is a gRPC service wrapper around an packer.
type Service struct {
	dealer    dealeri.Dealer
	finalizer *finalizer.Finalizer
}

var _ mbroker.ReadyToCreateDealsListener = (*Service)(nil)

// New returns a new Service.
func New(mb mbroker.MsgBroker, conf Config) (*Service, error) {
	fin := finalizer.NewFinalizer()

	h, err := libp2p.New(context.Background(),
		libp2p.ConnectionManager(connmgr.NewConnManager(500, 800, time.Minute)),
	)
	if err != nil {
		return nil, fmt.Errorf("creating host: %s", err)
	}

	if conf.RelayMaddr != "" {
		pinfo, err := peer.AddrInfoFromString(conf.RelayMaddr)
		if err != nil {
			return nil, fmt.Errorf("get addrinfo from relay multiaddr: %s", err)
		}
		ctx, cls := context.WithTimeout(context.Background(), time.Second*20)
		defer cls()
		if err := h.Connect(ctx, *pinfo); err != nil {
			cls()
			return nil, fmt.Errorf("connecting with relay: %s", err)
		}
		h.ConnManager().Protect(pinfo.ID, "relay")
		log.Debugf("connected with relay")
	}

	var lib dealeri.Dealer
	if conf.Mock {
		log.Warnf("running in mocked mode")
		lib = dealermock.New(mb, h, conf.RelayMaddr)
	} else {
		lotusClients, err := initializeLotusClients(fin, conf.LotusGatewayURL)
		if err != nil {
			return nil, err
		}

		opts := []filclient.Option{
			filclient.WithExportedKey(conf.LotusExportedWalletAddr),
			filclient.WithAllowUnverifiedDeals(conf.AllowUnverifiedDeals),
			filclient.WithMaxPriceLimits(conf.MaxVerifiedPricePerGiBPerEpoch, conf.MaxUnverifiedPricePerGiBPerEpoch),
		}
		if conf.RelayMaddr != "" {
			opts = append(opts, filclient.WithRelayAddr(conf.RelayMaddr))
		}
		filclient, err := filclient.New(&lotusClients[0], h, opts...)
		if err != nil {
			return nil, fin.Cleanupf("creating filecoin client: %s", err)
		}
		libi, err := dealer.New(conf.PostgresURI, mb, filclient)
		if err != nil {
			return nil, fin.Cleanupf("creating dealer: %v", err)
		}
		fin.Add(libi)
		lib = libi
	}

	s := &Service{
		dealer:    lib,
		finalizer: fin,
	}

	if err := mbroker.RegisterHandlers(mb, s); err != nil {
		return nil, fmt.Errorf("registering msgbroker handlers: %s", err)
	}

	return s, nil
}

func initializeLotusClients(fin *finalizer.Finalizer, gatewaysURLs []string) ([]v0api.FullNodeStruct, error) {
	lotusClients := []v0api.FullNodeStruct{}
	for _, url := range gatewaysURLs {
		var lotusAPI v0api.FullNodeStruct
		closer, err := jsonrpc.NewMergeClient(context.Background(), url, "Filecoin",
			[]interface{}{
				&lotusAPI.CommonStruct.Internal,
				&lotusAPI.NetStruct.Internal,
				&lotusAPI.Internal,
			},
			http.Header{},
		)

		if err != nil {
			log.Warnf("creating lotus gateway client: %s", err)
			continue
		}

		fin.Add(&nopCloser{closer})
		lotusClients = append(lotusClients, lotusAPI)
	}

	if len(lotusClients) == 0 {
		return []v0api.FullNodeStruct{}, errors.New("failed to instantiate Lotus client")
	}

	return lotusClients, nil
}

// OnReadyToCreateDeals process an event for deals to be executed.
func (s *Service) OnReadyToCreateDeals(ctx context.Context, ads dealeri.AuctionDeals) error {
	if err := s.dealer.ReadyToCreateDeals(ctx, ads); err != nil {
		if errors.Is(err, store.ErrAuctionDataExists) {
			log.Warnf("auction data with ID %s already processed, acking", ads.ID)
			return nil
		}
		return fmt.Errorf("processing ready to create deals: %s", err)
	}

	return nil
}

// Close the service.
func (s *Service) Close() error {
	defer log.Info("service was shutdown")
	return s.finalizer.Cleanup(nil)
}

type nopCloser struct {
	f func()
}

func (np *nopCloser) Close() error {
	np.f()
	return nil
}
