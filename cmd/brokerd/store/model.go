package store

import (
	"errors"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
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
		ID:      br.ID,
		DataCid: br.DataCid,
		Status:  br.Status,
		Metadata: metadata{
			Region: br.Metadata.Region,
		},
		StorageDealID: br.StorageDealID,
		CreatedAt:     br.CreatedAt,
		UpdatedAt:     br.UpdatedAt,
	}
}

func castToBrokerRequest(ibr brokerRequest) broker.BrokerRequest {
	return broker.BrokerRequest{
		ID:      ibr.ID,
		DataCid: ibr.DataCid,
		Status:  ibr.Status,
		Metadata: broker.Metadata{
			Region: ibr.Metadata.Region,
		},
		StorageDealID: ibr.StorageDealID,
		CreatedAt:     ibr.CreatedAt,
		UpdatedAt:     ibr.UpdatedAt,
	}
}
