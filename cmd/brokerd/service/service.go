package service

import (
	"fmt"
	"strings"
)

type Config struct {
	GrpcListenAddress string
}

type Service struct {
	config Config
}

func New(config Config) (*Service, error) {
	s := &Service{
		config: config,
	}

	return s, nil
}

func (s *Service) Close() error {
	var errors []string

	// TODO: close gRPC server when exists

	if errors != nil {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}
