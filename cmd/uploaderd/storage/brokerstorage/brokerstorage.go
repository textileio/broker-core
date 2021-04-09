package brokerstorage

import (
	"context"
	"fmt"
	"io"

	"github.com/textileio/broker-core/cmd/uploaderd/storage"
	"github.com/textileio/broker-core/cmd/uploaderd/storage/brokerstorage/auth"
	"github.com/textileio/broker-core/cmd/uploaderd/storage/brokerstorage/broker"
	"github.com/textileio/broker-core/cmd/uploaderd/storage/brokerstorage/uploader"
)

type BrokerStorage struct {
	auth   auth.Authorizer
	up     uploader.Uploader
	broker broker.Broker
}

var _ storage.Storage = (*BrokerStorage)(nil)

func New(auth auth.Authorizer, up uploader.Uploader, broker broker.Broker) (*BrokerStorage, error) {
	return &BrokerStorage{
		auth:   auth,
		up:     up,
		broker: broker,
	}, nil

}

func (bs *BrokerStorage) IsStorageAuthorized(ctx context.Context, identity string) (bool, string, error) {
	return bs.auth.IsAuthorized(identity)
}

// CreateStorageRequest creates a StorageRequest using data from a stream.
func (bs *BrokerStorage) CreateStorageRequest(ctx context.Context, r io.Reader, meta storage.Metadata) (storage.StorageRequest, error) {
	c, err := bs.up.Store(ctx, r)
	if err != nil {
		return storage.StorageRequest{}, fmt.Errorf("storing stream: %s", err)
	}

	sr, err := bs.broker.CreateStorageRequest(ctx, c, meta)
	if err != nil {
		return storage.StorageRequest{}, fmt.Errorf("creating storage request: %s", err)
	}

	return sr, nil
}
