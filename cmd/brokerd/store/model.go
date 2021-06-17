package store

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/broker"
)

type brokerRequest struct {
	ID            broker.BrokerRequestID
	DataCid       cid.Cid
	Status        broker.BrokerRequestStatus
	Metadata      metadata
	StorageDealID broker.StorageDealID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type metadata struct {
	Region string `json:"region"`
}

func (br brokerRequest) validate() error {
	if br.ID == "" {
		return errors.New("id is empty")
	}
	if !br.DataCid.Defined() {
		return errors.New("datacid is undefined")
	}
	if err := br.Metadata.validate(); err != nil {
		return fmt.Errorf("invalid metadata: %s", err)
	}

	return nil
}

func (m metadata) validate() error {
	return nil
}

func castToInternalBrokerRequest(br broker.BrokerRequest) brokerRequest {
	return brokerRequest{
		ID:            br.ID,
		DataCid:       br.DataCid,
		Status:        br.Status,
		StorageDealID: br.StorageDealID,
		CreatedAt:     br.CreatedAt,
		UpdatedAt:     br.UpdatedAt,
	}
}

func castToBrokerRequest(ibr brokerRequest) broker.BrokerRequest {
	return broker.BrokerRequest{
		ID:            ibr.ID,
		DataCid:       ibr.DataCid,
		Status:        ibr.Status,
		StorageDealID: ibr.StorageDealID,
		CreatedAt:     ibr.CreatedAt,
		UpdatedAt:     ibr.UpdatedAt,
	}
}

type storageDeal struct {
	ID               broker.StorageDealID
	Status           broker.StorageDealStatus
	BrokerRequestIDs []broker.BrokerRequestID
	RepFactor        int
	DealDuration     int
	Sources          sources
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Error            string

	PayloadCid cid.Cid

	PieceCid  cid.Cid
	PieceSize uint64

	Deals []minerDeal
}

type sources struct {
	CARURL  *string
	CARIPFS *carIPFS
}

type carIPFS struct {
	Cid        string
	Multiaddrs []string
}

type minerDeal struct {
	StorageDealID broker.StorageDealID
	AuctionID     broker.AuctionID
	BidID         broker.BidID
	Miner         string
	CreatedAt     time.Time
	UpdatedAt     time.Time

	DealID         int64
	DealExpiration uint64
	ErrorCause     string
}

func castToStorageDeal(isd storageDeal) (broker.StorageDeal, error) {
	bd := broker.StorageDeal{
		ID:               isd.ID,
		Status:           isd.Status,
		BrokerRequestIDs: make([]broker.BrokerRequestID, len(isd.BrokerRequestIDs)),
		RepFactor:        isd.RepFactor,
		DealDuration:     isd.DealDuration,
		CreatedAt:        isd.CreatedAt,
		UpdatedAt:        isd.UpdatedAt,
		Error:            isd.Error,

		PayloadCid: isd.PayloadCid,

		PieceCid:  isd.PieceCid,
		PieceSize: isd.PieceSize,

		Deals: make([]broker.MinerDeal, len(isd.Deals)),
	}
	copy(bd.BrokerRequestIDs, isd.BrokerRequestIDs)

	for i := range isd.Deals {
		bd.Deals[i] = broker.MinerDeal{
			StorageDealID: isd.Deals[i].StorageDealID,
			AuctionID:     isd.Deals[i].AuctionID,
			BidID:         isd.Deals[i].BidID,
			CreatedAt:     isd.Deals[i].CreatedAt,
			UpdatedAt:     isd.Deals[i].UpdatedAt,

			Miner:          isd.Deals[i].Miner,
			DealID:         isd.Deals[i].DealID,
			DealExpiration: isd.Deals[i].DealExpiration,
			ErrorCause:     isd.Deals[i].ErrorCause,
		}
	}

	if isd.Sources.CARURL != nil {
		u, err := url.Parse(*isd.Sources.CARURL)
		if err != nil {
			return broker.StorageDeal{}, fmt.Errorf("parsing url: %s", err)
		}
		bd.Sources.CARURL = &broker.CARURL{URL: *u}
	}
	if isd.Sources.CARIPFS != nil {
		carCID, err := cid.Parse(isd.Sources.CARIPFS.Cid)
		if err != nil {
			return broker.StorageDeal{}, fmt.Errorf("parsing cid: %s", err)
		}
		multiaddrs := make([]multiaddr.Multiaddr, len(isd.Sources.CARIPFS.Multiaddrs))
		for i, strmaddr := range isd.Sources.CARIPFS.Multiaddrs {
			maddr, err := multiaddr.NewMultiaddr(strmaddr)
			if err != nil {
				return broker.StorageDeal{}, fmt.Errorf("parsing multiaddr: %s", err)
			}
			multiaddrs[i] = maddr
		}
		bd.Sources.CARIPFS = &broker.CARIPFS{
			Cid:        carCID,
			Multiaddrs: multiaddrs,
		}
	}

	return bd, nil
}
