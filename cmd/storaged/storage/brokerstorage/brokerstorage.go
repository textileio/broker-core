package brokerstorage

import (
	"context"
	"fmt"
	"io"

	"github.com/textileio/broker-core/auth"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/storaged/storage"
	"github.com/textileio/broker-core/cmd/storaged/storage/brokerstorage/uploader"
)

// BrokerStorage its an API implementation of the storage service.
type BrokerStorage struct {
	auth   auth.Authorizer
	up     uploader.Uploader
	broker broker.BrokerRequestor
}

var _ storage.Requester = (*BrokerStorage)(nil)

// New returns a new BrokerStorage.
func New(auth auth.Authorizer, up uploader.Uploader, broker broker.BrokerRequestor) (*BrokerStorage, error) {
	return &BrokerStorage{
		auth:   auth,
		up:     up,
		broker: broker,
	}, nil
}

// IsAuthorized resolves if the provided identity is authorized to use the
// service. If that isn't the case, a string is also return to exply why.
func (bs *BrokerStorage) IsAuthorized(ctx context.Context, identity string) (bool, string, error) {
	return bs.auth.IsAuthorized(ctx, identity)
}

// CreateFromReader creates a StorageRequest using data from a stream.
func (bs *BrokerStorage) CreateFromReader(
	ctx context.Context,
	r io.Reader,
	meta storage.Metadata,
) (storage.Request, error) {
	c, err := bs.up.Store(ctx, r)
	if err != nil {
		return storage.Request{}, fmt.Errorf("storing stream: %s", err)
	}

	brokerMeta := broker.Metadata{
		Region: meta.Region,
	}
	sr, err := bs.broker.Create(ctx, c, brokerMeta)
	if err != nil {
		return storage.Request{}, fmt.Errorf("creating storage request: %s", err)
	}
	status, err := brokerRequestStatusToStorageRequestStatus(sr.Status)
	if err != nil {
		return storage.Request{}, fmt.Errorf("mapping statuses: %s", err)
	}

	return storage.Request{
		ID:         string(sr.ID),
		Cid:        c,
		StatusCode: status,
	}, nil
}

// Get returns a Request by id.
func (bs *BrokerStorage) Get(ctx context.Context, id string) (storage.Request, error) {
	br, err := bs.broker.Get(ctx, broker.BrokerRequestID(id))
	if err != nil {
		return storage.Request{}, fmt.Errorf("getting broker request: %s", err)
	}

	status, err := brokerRequestStatusToStorageRequestStatus(br.Status)
	if err != nil {
		return storage.Request{}, fmt.Errorf("mapping statuses: %s", err)
	}
	sr := storage.Request{
		ID:         string(br.ID),
		Cid:        br.DataCid,
		StatusCode: status,
	}

	return sr, nil
}

func brokerRequestStatusToStorageRequestStatus(status broker.BrokerRequestStatus) (storage.Status, error) {
	switch status {
	case broker.RequestBatching:
		return storage.StatusBatching, nil
	case broker.RequestPreparing:
		return storage.StatusPreparing, nil
	case broker.RequestAuctioning:
		return storage.StatusAuctioning, nil
	case broker.RequestDealMaking:
		return storage.StatusDealMaking, nil
	case broker.RequestSuccess:
		return storage.StatusSuccess, nil
	default:
		return storage.StatusUnknown, fmt.Errorf("unknown status: %s", status)
	}
}
