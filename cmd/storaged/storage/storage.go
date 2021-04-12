package storage

import (
	"context"
	"io"

	"github.com/ipfs/go-cid"
)

// StorageRequester contains handles raw-files uploads of data to be
// stored with the Broker service.
type StorageRequester interface {
	IsAuthorized(ctx context.Context, identity string) (bool, string, error)
	CreateFromReader(ctx context.Context, r io.Reader, meta Metadata) (StorageRequest, error)
	// Potentially other future methods here to cover other use-cases like:
	// - CreateFromCid(context.Context, cid.Cid, Metadata) // Fetch from IPFS network
	// - CreateFromRemote(context.Context, RemoteOrigin, Metadata) // Fetch from some remote origin.
}

// Metadata contains extra data to be considered by the Broker.
type Metadata struct {
	Region string
}

// StorageRequestStatus is the status of a StorageRequest
type StorageRequestStatus int

const (
	// StatusUnknown is the default value to an unitialized
	// StorageRequest. This status must be considered invalid in any
	// real StorageRequest instance.
	StatusUnknown StorageRequestStatus = iota
	StatusFetchingData
	StatusBatching
)

// StorageRequest is a request for storing data in a Broker.
type StorageRequest struct {
	ID         string               `json:"id"`
	Cid        cid.Cid              `json:"cid"`
	StatusCode StorageRequestStatus `json:"status_code"`
}
