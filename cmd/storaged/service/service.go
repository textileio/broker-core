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
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/uploader/ipfsuploader"
	"github.com/textileio/broker-core/rpc"
)

// Config provides configuration parameters for a service.
type Config struct {
	HTTPListenAddr        string
	UploaderIPFSMultiaddr string
	BrokerAPIAddr         string
	AuthAddr              string
	SkipAuth              bool
	IpfsMultiaddrs        []multiaddr.Multiaddr
	PinataJWT             string
	BearerTokens          []string
	MaxUploadSize         uint
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
	server, err := httpapi.NewServer(config.HTTPListenAddr, config.SkipAuth, storage, config.MaxUploadSize)
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
	auth, err := brokerauth.New(config.AuthAddr, config.BearerTokens)
	if err != nil {
		return nil, fmt.Errorf("creating broker auth: %s", err)
	}

	up, err := ipfsuploader.New(config.UploaderIPFSMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("creating broker uploader: %s", err)
	}

	client, err := client.New(config.BrokerAPIAddr, rpc.GetClientOpts(config.BrokerAPIAddr)...)
	if err != nil {
		return nil, fmt.Errorf("creating brokerd gRPC client: %s", err)
	}

	bs, err := brokerstorage.New(auth, up, client, config.IpfsMultiaddrs, config.PinataJWT)
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
