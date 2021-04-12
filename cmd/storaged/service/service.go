package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/textileio/broker-core/cmd/storaged/httpapi"
	"github.com/textileio/broker-core/cmd/storaged/storage"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/brokerauth"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/texbroker"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/uploader/ipfsuploader"
)

type Config struct {
	HttpListenAddr        string
	UploaderIPFSMultiaddr string
}

type Service struct {
	config Config

	httpAPIServer *http.Server
}

func New(config Config) (*Service, error) {
	storage, err := createStorage(config)
	if err != nil {
		return nil, fmt.Errorf("creating storage component: %s", err)
	}

	// Bootstrap HTTP API server.
	httpAPIServer, err := httpapi.NewServer(config.HttpListenAddr, storage)
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

func createStorage(config Config) (storage.StorageRequester, error) {
	auth, err := brokerauth.New()
	if err != nil {
		return nil, fmt.Errorf("creating broker auth: %s", err)
	}

	up, err := ipfsuploader.New(config.UploaderIPFSMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("creating broker uploader: %s", err)
	}

	brok, err := texbroker.New()
	if err != nil {
		return nil, fmt.Errorf("creating broker service: %s", err)
	}

	bs, err := brokerstorage.New(auth, up, brok)
	if err != nil {
		return nil, fmt.Errorf("creating broker storage: %s", err)
	}

	return bs, nil
}

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
