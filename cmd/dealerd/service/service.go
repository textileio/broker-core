package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/textileio/bidbot/lib/dshelper"
	"github.com/textileio/bidbot/lib/finalizer"
	"github.com/textileio/broker-core/cmd/dealerd/dealer"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/filclient"
	"github.com/textileio/broker-core/cmd/dealerd/dealermock"
	dealeri "github.com/textileio/broker-core/dealer"
	mbroker "github.com/textileio/broker-core/msgbroker"
	golog "github.com/textileio/go-log/v2"
)

var log = golog.Logger("dealer/service")

// Config defines params for Service configuration.
type Config struct {
	MongoDBName string
	MongoURI    string

	LotusGatewayURL         string
	LotusExportedWalletAddr string

	AllowUnverifiedDeals             bool
	MaxVerifiedPricePerGiBPerEpoch   int64
	MaxUnverifiedPricePerGiBPerEpoch int64

	Mock bool
}

// Service is a gRPC service wrapper around an packer.
type Service struct {
	dealer    dealeri.Dealer
	finalizer *finalizer.Finalizer
}

// New returns a new Service.
func New(mb mbroker.MsgBroker, conf Config) (*Service, error) {
	if err := validateConfig(conf); err != nil {
		return nil, fmt.Errorf("config is invalid: %s", err)
	}

	fin := finalizer.NewFinalizer()

	ds, err := dshelper.NewMongoTxnDatastore(conf.MongoURI, conf.MongoDBName)
	if err != nil {
		return nil, fmt.Errorf("creating datastore: %s", err)
	}
	fin.Add(ds)

	var lib dealeri.Dealer
	if conf.Mock {
		log.Warnf("running in mocked mode")
		lib = dealermock.New(mb)
	} else {
		var lotusAPI v0api.FullNodeStruct
		closer, err := jsonrpc.NewMergeClient(context.Background(), conf.LotusGatewayURL, "Filecoin",
			[]interface{}{
				&lotusAPI.CommonStruct.Internal,
				&lotusAPI.Internal,
			},
			http.Header{},
		)
		if err != nil {
			return nil, fin.Cleanupf("creating lotus gateway client: %s", err)
		}
		fin.Add(&nopCloser{closer})

		filclient, err := filclient.New(
			&lotusAPI,
			filclient.WithExportedKey(conf.LotusExportedWalletAddr),
			filclient.WithAllowUnverifiedDeals(conf.AllowUnverifiedDeals),
			filclient.WithMaxPriceLimits(conf.MaxVerifiedPricePerGiBPerEpoch, conf.MaxUnverifiedPricePerGiBPerEpoch),
		)
		if err != nil {
			return nil, fin.Cleanupf("creating filecoin client: %s", err)
		}
		libi, err := dealer.New(ds, mb, filclient)
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

// OnReadyToCreateDeals process an event for deals to be executed.
func (s *Service) OnReadyToCreateDeals(ctx context.Context, ads dealeri.AuctionDeals) error {
	if err := s.dealer.ReadyToCreateDeals(ctx, ads); err != nil {
		return fmt.Errorf("processing ready to create deals: %s", err)
	}

	return nil
}

// Close the service.
func (s *Service) Close() error {
	defer log.Info("service was shutdown")
	return s.finalizer.Cleanup(nil)
}

func validateConfig(conf Config) error {
	if conf.MongoDBName == "" {
		return errors.New("mongo db name is empty")
	}
	if conf.MongoURI == "" {
		return errors.New("mongo uri is empty")
	}

	return nil
}

type nopCloser struct {
	f func()
}

func (np *nopCloser) Close() error {
	np.f()
	return nil
}
