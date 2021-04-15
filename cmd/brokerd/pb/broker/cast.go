package broker

import (
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

func FromProtoBrokerRequest(brproto *BR) (broker.BrokerRequest, error) {
	c, err := cid.Decode(brproto.Cid)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("decoding cid: %s", err)
	}

	var status broker.BrokerRequestStatus
	switch brproto.Status {
	case BrokerRequestStatus_UNSPECIFIED:
		status = broker.BrokerRequestUnknown
	case BrokerRequestStatus_BATCHING:
		status = broker.BrokerRequestBatching
	case BrokerRequestStatus_PREPARING:
		status = broker.BrokerRequestPreparing
	case BrokerRequestStatus_AUCTIONING:
		status = broker.BrokerRequestAuctioning
	case BrokerRequestStatus_DEALMAKING:
		status = broker.BrokerRequestDealMaking
	case BrokerRequestStatus_SUCCESS:
		status = broker.BrokerRequestSuccess
	}

	var metadata broker.Metadata
	if brproto.Meta != nil {
		metadata.Region = brproto.Meta.Region
	}

	br := broker.BrokerRequest{
		ID:            broker.BrokerRequestID(brproto.Id),
		DataCid:       c,
		Status:        status,
		Metadata:      metadata,
		StorageDealID: broker.StorageDealID(brproto.StorageDealId),
		CreatedAt:     brproto.CreatedAt.AsTime(),
		UpdatedAt:     brproto.UpdatedAt.AsTime(),
	}

	return br, nil
}
