package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/storaged/httpapi"
	"github.com/textileio/broker-core/cmd/storaged/storage"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/brokerauth"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/texbroker"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/uploader/ipfsuploader"
	"github.com/textileio/broker-core/rpc"
)

// Config provides configuration parameters for a service.
type Config struct {
	// HTTPListenAddr is the binding address for the public REST API.
	HTTPListenAddr string
	// UploaderIPFSMultiaddr is the multiaddress of the IPFS layer.
	UploaderIPFSMultiaddr string
	// BrokerAPIAddr is the Broker API address.
	BrokerAPIAddr string
	// AuthdAddr is the address of authd.
	AuthdAddr string
}

// Service provides an implementation of the Storage API.
type Service struct {
	config Config

	httpAPIServer *http.Server
}

// New returns a new Service.
func New(config Config) (*Service, error) {
	storage, err := createStorage(config)
	if err != nil {
		return nil, fmt.Errorf("creating storage component: %s", err)
	}

	// Bootstrap HTTP API server.
	httpAPIServer, err := httpapi.NewServer(config.HTTPListenAddr, storage)
	if err != nil {
		return nil, fmt.Errorf("creating http server: %s", err)
	}

	// Generate service.
	s := &Service{
		config: config,

		httpAPIServer: httpAPIServer,
	}
	return s, nil
}

func createStorage(config Config) (storage.Requester, error) {
	auth, err := brokerauth.New(config.AuthdAddr)
	if err != nil {
		return nil, fmt.Errorf("creating broker auth: %s", err)
	}

	up, err := ipfsuploader.New(config.UploaderIPFSMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("creating broker uploader: %s", err)
	}

	brokerdClient, err := client.New(config.BrokerAPIAddr, rpc.GetClientOpts(config.BrokerAPIAddr)...)
	if err != nil {
		return nil, fmt.Errorf("creating brokerd gRPC client: %s", err)
	}

	brok, err := texbroker.New(brokerdClient)
	if err != nil {
		return nil, fmt.Errorf("creating broker service: %s", err)
	}

	bs, err := brokerstorage.New(auth, up, brok)
	if err != nil {
		return nil, fmt.Errorf("creating broker storage: %s", err)
	}

	return bs, nil
}

// Close gracefully closes the service.
func (s *Service) Close() error {
	var errors []string

	if err := s.httpAPIServer.Close(); err != nil {
		errors = append(errors, fmt.Sprintf("closing http api server: %s", err))
	}

	if errors != nil {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}
	return nil
}
