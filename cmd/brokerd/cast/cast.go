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
	case pb.BrokerRequest_ERROR:
		status = broker.RequestError
	default:
		return broker.BrokerRequest{}, fmt.Errorf("unknown status: %s", brproto.Status)
	}

	return broker.BrokerRequest{
		ID:            broker.BrokerRequestID(brproto.Id),
		DataCid:       c,
		Status:        status,
		StorageDealID: broker.StorageDealID(brproto.StorageDealId),
		CreatedAt:     brproto.CreatedAt.AsTime(),
		UpdatedAt:     brproto.UpdatedAt.AsTime(),
	}, nil
}

// FromProtoBrokerRequestInfo transforms a pb.BrokerRequest to broker.BrokerRequest.
func FromProtoBrokerRequestInfo(brproto *pb.GetBrokerRequestInfoResponse) (broker.BrokerRequestInfo, error) {
	br, err := FromProtoBrokerRequest(brproto.BrokerRequest)
	if err != nil {
		return broker.BrokerRequestInfo{}, nil
	}

	bri := broker.BrokerRequestInfo{
		BrokerRequest: br,
		Deals:         make([]broker.BrokerRequestDeal, len(brproto.Deals)),
	}
	for i, d := range brproto.Deals {
		deal := broker.BrokerRequestDeal{
			Miner:      d.Miner,
			DealID:     d.DealId,
			Expiration: d.Expiration,
		}
		bri.Deals[i] = deal
	}

	return bri, nil
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
	case broker.RequestError:
		pbStatus = pb.BrokerRequest_ERROR
	default:
		return nil, fmt.Errorf("unknown status: %d", br.Status)
	}

	return &pb.BrokerRequest{
		Id:            string(br.ID),
		DataCid:       br.DataCid.String(),
		Status:        pbStatus,
		StorageDealId: string(br.StorageDealID),
		CreatedAt:     timestamppb.New(br.CreatedAt),
		UpdatedAt:     timestamppb.New(br.UpdatedAt),
	}, nil
}

// BrokerRequestInfoToProto maps a broker.
func BrokerRequestInfoToProto(br broker.BrokerRequestInfo) (*pb.GetBrokerRequestInfoResponse, error) {
	protobr, err := BrokerRequestToProto(br.BrokerRequest)
	if err != nil {
		return nil, fmt.Errorf("creating proto for broker request: %s", err)
	}

	deals := make([]*pb.GetBrokerRequestInfoResponse_BrokerRequestDeal, len(br.Deals))
	for i, d := range br.Deals {
		deal := &pb.GetBrokerRequestInfoResponse_BrokerRequestDeal{
			Miner:      d.Miner,
			DealId:     d.DealID,
			Expiration: d.Expiration,
		}
		deals[i] = deal
	}

	return &pb.GetBrokerRequestInfoResponse{
		BrokerRequest: protobr,
		Deals:         deals,
	}, nil
}
