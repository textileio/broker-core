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
	CreateFromReader(ctx context.Context, r io.Reader) (Request, error)
	CreateFromExternalSource(ctx context.Context, adr AuctionDataRequest) (Request, error)
	GetCARHeader(ctx context.Context, c cid.Cid, w io.Writer) (bool, error)
	GetCAR(ctx context.Context, c cid.Cid, w io.Writer) (bool, error)
	GetRequestInfo(ctx context.Context, id string) (RequestInfo, error)
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

// RequestInfo describes the current state of a request.
type RequestInfo struct {
	Request Request `json:"request"`
	Deals   []Deal  `json:"deals"`
}

// Deal contains information of an on-chain deal.
type Deal struct {
	StorageProviderID string `json:"storage_provider_id"`
	DealID            int64  `json:"deal_id"`
	Expiration        uint64 `json:"deal_expiration"`
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
	Cid        string   `json:"cid"`
	Multiaddrs []string `json:"multiaddrs"`
}
