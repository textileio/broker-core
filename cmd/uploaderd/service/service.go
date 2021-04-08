package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/textileio/broker-core/cmd/uploaderd/httpapi"
)

type Config struct {
	HttpListenAddr string
}

type Service struct {
	config Config

	httpAPIServer *http.Server
}

func New(config Config) (*Service, error) {
	// Bootstrap HTTP API server.
	httpAPIServer, err := httpapi.NewServer(config.HttpListenAddr)
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
