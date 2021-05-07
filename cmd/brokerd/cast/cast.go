package cast

import (
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FromProtoBrokerRequest transforms a pb.BrokerRequest to broker.BrokerRequest.
func FromProtoBrokerRequest(brproto *pb.BrokerRequest) (broker.BrokerRequest, error) {
	c, err := cid.Decode(brproto.DataCid)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("decoding cid: %s", err)
	}

	var status broker.BrokerRequestStatus
	switch brproto.Status {
	case pb.BrokerRequest_UNSPECIFIED:
		status = broker.RequestUnknown
	case pb.BrokerRequest_BATCHING:
		status = broker.RequestBatching
	case pb.BrokerRequest_PREPARING:
		status = broker.RequestPreparing
	case pb.BrokerRequest_AUCTIONING:
		status = broker.RequestAuctioning
	case pb.BrokerRequest_DEALMAKING:
		status = broker.RequestDealMaking
	case pb.BrokerRequest_SUCCESS:
		status = broker.RequestSuccess
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

// BrokerRequestToProto maps a broker.BrokerRequest to pb.BrokerRequest.
func BrokerRequestToProto(br broker.BrokerRequest) (*pb.BrokerRequest, error) {
	var pbStatus pb.BrokerRequest_Status
	switch br.Status {
	case broker.RequestUnknown:
		pbStatus = pb.BrokerRequest_UNSPECIFIED
	case broker.RequestBatching:
		pbStatus = pb.BrokerRequest_BATCHING
	case broker.RequestPreparing:
		pbStatus = pb.BrokerRequest_PREPARING
	case broker.RequestAuctioning:
		pbStatus = pb.BrokerRequest_AUCTIONING
	case broker.RequestDealMaking:
		pbStatus = pb.BrokerRequest_DEALMAKING
	case broker.RequestSuccess:
		pbStatus = pb.BrokerRequest_SUCCESS
	default:
		return nil, fmt.Errorf("unknown status: %d", br.Status)
	}

	return &pb.BrokerRequest{
		Id:      string(br.ID),
		DataCid: br.DataCid.String(),
		Status:  pbStatus,
		Meta: &pb.BrokerRequest_Metadata{
			Region: br.Metadata.Region,
		},
		StorageDealId: string(br.StorageDealID),
		CreatedAt:     timestamppb.New(br.CreatedAt),
		UpdatedAt:     timestamppb.New(br.UpdatedAt),
	}, nil
}

// FinalizedDealsToPb casts broker.FinalizedAuctionDeal to its proto representation.
func FinalizedDealsToPb(fads []broker.FinalizedAuctionDeal) []*pb.StorageDealFinalizedDealsRequest_FinalizedDeal {
	res := make([]*pb.StorageDealFinalizedDealsRequest_FinalizedDeal, len(fads))
	for i, fad := range fads {
		res[i] = &pb.StorageDealFinalizedDealsRequest_FinalizedDeal{
			StorageDealId:  string(fad.StorageDealID),
			DealId:         fad.DealID,
			DealExpiration: fad.DealExpiration,
			ErrorCause:     fad.ErrorCause,
		}
	}

	return res
}
