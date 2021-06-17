package storage

import (
	"context"
	"io"

	"github.com/ipfs/go-cid"
)

// Requester contains handles raw-files uploads of data to be
// stored with the Broker service.
type Requester interface {
	IsAuthorized(ctx context.Context, identity string) (bool, string, error)
	CreateFromReader(ctx context.Context, r io.Reader, meta Metadata) (Request, error)
	CreateFromExternalSource(ctx context.Context, adr AuctionDataRequest) (Request, error)
	GetCAR(ctx context.Context, c cid.Cid, w io.Writer) error
	Get(ctx context.Context, id string) (Request, error)
}

// Metadata contains extra data to be considered by the Broker.
type Metadata struct {
	Region string
}

// Status is the status of a StorageRequest.
type Status int

const (
	// StatusUnknown is the default value to an unitialized
	// StorageRequest. This status must be considered invalid in any
	// real StorageRequest instance.
	StatusUnknown Status = iota
	// StatusBatching indicates that the storage request is being batched.
	StatusBatching
	// StatusPreparing indicates that the batch containing the data is being prepared.
	StatusPreparing
	// StatusAuctioning indicates that the batch containing the data is being auctioned.
	StatusAuctioning
	// StatusDealMaking indicates that the data is in deal-making process.
	StatusDealMaking
	// StatusSuccess indicates that the request was stored in Filecoin.
	StatusSuccess
)

// Request is a request for storing data in a Broker.
type Request struct {
	ID         string  `json:"id"`
	Cid        cid.Cid `json:"cid"`
	StatusCode Status  `json:"status_code"`
}

// AuctionDataRequest contains information about a prepared dataset hosted externally.
type AuctionDataRequest struct {
	PayloadCid string   `json:"payloadCid"`
	PieceCid   string   `json:"pieceCid"`
	PieceSize  uint64   `json:"pieceSize"`
	RepFactor  int      `json:"repFactor"`
	Deadline   string   `json:"deadline"`
	CARURL     *CARURL  `json:"carURL"`
	CARIPFS    *CARIPFS `json:"carIPFS"`
}

// CARURL contains details of a CAR file stored in an HTTP endpoint.
type CARURL struct {
	URL string `json:"url"`
}

// CARIPFS contains details of a CAR file Cid stored in an HTTP endpoint.
type CARIPFS struct {
	Cid            string   `json:"cid"`
	NodesMultiaddr []string `json:"nodesMultiaddr"`
}
