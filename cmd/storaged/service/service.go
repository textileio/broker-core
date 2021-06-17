package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/storaged/httpapi"
	"github.com/textileio/broker-core/cmd/storaged/storage"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/brokerauth"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/texbroker"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/uploader/ipfsuploader"
	"github.com/textileio/broker-core/rpc"
)

// TODO(jsign): move to options.
// Config provides configuration parameters for a service.
type Config struct {
	// HTTPListenAddr is the binding address for the public REST API.
	HTTPListenAddr string
	// UploaderIPFSMultiaddr is the multiaddress of the IPFS layer.
	UploaderIPFSMultiaddr string
	// BrokerAPIAddr is the Broker API address.
	BrokerAPIAddr string
	// AuthAddr is the address of authd.
	AuthAddr string
	// SkipAuth disables authorizations checks.
	SkipAuth bool
	// IpfsMultiaddrs provides a complete set of ipfs nodes APIs to make retrievals of pinned data.
	IpfsMultiaddrs []multiaddr.Multiaddr
}

// Service provides an implementation of the Storage API.
type Service struct {
	config Config

	httpServer *http.Server
}

// New returns a new Service.
func New(config Config) (*Service, error) {
	storage, err := createStorage(config)
	if err != nil {
		return nil, fmt.Errorf("creating storage component: %s", err)
	}

	// Bootstrap HTTP API server.
	server, err := httpapi.NewServer(config.HTTPListenAddr, config.SkipAuth, storage)
	if err != nil {
		return nil, fmt.Errorf("creating http server: %s", err)
	}

	// Generate service.
	s := &Service{
		config:     config,
		httpServer: server,
	}

	return s, nil
}

func createStorage(config Config) (storage.Requester, error) {
	auth, err := brokerauth.New(config.AuthAddr)
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

	bs, err := brokerstorage.New(auth, up, brok, config.IpfsMultiaddrs)
	if err != nil {
		return nil, fmt.Errorf("creating broker storage: %s", err)
	}

	return bs, nil
}

// Close gracefully closes the service.
func (s *Service) Close() error {
	var errors []string

	if err := s.httpServer.Close(); err != nil {
		errors = append(errors, fmt.Sprintf("closing http api server: %s", err))
	}

	if errors != nil {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}
	return nil
}
