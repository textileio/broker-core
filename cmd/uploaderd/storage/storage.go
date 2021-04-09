package storage

import (
	"context"
	"io"
)

// Storage contains handles raw-files uploads of data to be
// stored with the Broker service.
type Storage interface {
	IsStorageAuthorized(ctx context.Context, identity string) (bool, string, error)
	CreateStorageRequest(ctx context.Context, r io.Reader, meta Metadata) (StorageRequest, error)
}

// StorageRequestStatus is the status of a StorageRequest
type StorageRequestStatus int

const (
	// StorageRequestUnkown is the default value to an unitialized
	// StorageRequest. This status must be considered invalid in any
	// real StorageRequest instance.
	StorageRequestUnkown StorageRequestStatus = iota
	StorageRequestStatusPendingPrepare
	// TODO: There's a high chance that the status shouldn't be
	// needed here. This daemon only create StorageRequests, so
	// doesn't need to know all status codes. This can simplify
	// having things up to date with the Broker which is the source of truth.
	// If for some reason we need all status here, we should move
	// this statuses to some shared library.
)

// StorageRequest is a request for storing data in a Broker.
type StorageRequest struct {
	ID         string               `json:"id"`
	StatusCode StorageRequestStatus `json:"status_code"`
}

// Metadata contains extra data to be considered by the Broker.
type Metadata struct {
	Region string
}
