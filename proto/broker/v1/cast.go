package v1

import (
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"

	pb "github.com/textileio/broker-core/gen/broker/v1"
)

func FromProtoBrokerRequest(brproto *pb.BrokerRequest) (broker.BrokerRequest, error) {
	c, err := cid.Decode(brproto.DataCid)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("decoding cid: %s", err)
	}

	var status broker.BrokerRequestStatus
	switch brproto.Status {
	case pb.BrokerRequestStatus_UNSPECIFIED:
		status = broker.RequestUnknown
	case pb.BrokerRequestStatus_BATCHING:
		status = broker.RequestBatching
	case pb.BrokerRequestStatus_PREPARING:
		status = broker.RequestPreparing
	case pb.BrokerRequestStatus_AUCTIONING:
		status = broker.RequestAuctioning
	case pb.BrokerRequestStatus_DEALMAKING:
		status = broker.RequestDealMaking
	case pb.BrokerRequestStatus_SUCCESS:
		status = broker.BrokerRequestSuccess
	default:
		return broker.BrokerRequest{}, fmt.Errorf("unknown status: %s", brproto.Status)
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
