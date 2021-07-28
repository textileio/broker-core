package cast

import (
	"errors"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FromProtoStorageRequest transforms a pb.StorageRequest to broker.StorageRequest.
func FromProtoStorageRequest(brproto *pb.StorageRequest) (broker.StorageRequest, error) {
	dataCid, err := cid.Cast(brproto.DataCid)
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("decoding cid: %s", err)
	}
	if !dataCid.Defined() {
		return broker.StorageRequest{}, errors.New("data cid is undefined")
	}

	var status broker.StorageRequestStatus
	switch brproto.Status {
	case pb.StorageRequest_UNSPECIFIED:
		status = broker.RequestUnknown
	case pb.StorageRequest_BATCHING:
		status = broker.RequestBatching
	case pb.StorageRequest_PREPARING:
		status = broker.RequestPreparing
	case pb.StorageRequest_AUCTIONING:
		status = broker.RequestAuctioning
	case pb.StorageRequest_DEALMAKING:
		status = broker.RequestDealMaking
	case pb.StorageRequest_SUCCESS:
		status = broker.RequestSuccess
	case pb.StorageRequest_ERROR:
		status = broker.RequestError
	default:
		return broker.StorageRequest{}, fmt.Errorf("unknown status: %s", brproto.Status)
	}

	return broker.StorageRequest{
		ID:        broker.StorageRequestID(brproto.Id),
		DataCid:   dataCid,
		Status:    status,
		Origin:    brproto.Origin,
		BatchID:   broker.BatchID(brproto.BatchId),
		CreatedAt: brproto.CreatedAt.AsTime(),
		UpdatedAt: brproto.UpdatedAt.AsTime(),
	}, nil
}

// FromProtoStorageRequestInfo transforms a pb.StorageRequest to broker.StorageRequest.
func FromProtoStorageRequestInfo(brproto *pb.GetStorageRequestInfoResponse) (broker.StorageRequestInfo, error) {
	br, err := FromProtoStorageRequest(brproto.StorageRequest)
	if err != nil {
		return broker.StorageRequestInfo{}, nil
	}

	bri := broker.StorageRequestInfo{
		StorageRequest: br,
		Deals:          make([]broker.StorageRequestDeal, len(brproto.Deals)),
	}
	for i, d := range brproto.Deals {
		deal := broker.StorageRequestDeal{
			StorageProviderID: d.StorageProviderId,
			DealID:            d.DealId,
			Expiration:        d.Expiration,
		}
		bri.Deals[i] = deal
	}

	return bri, nil
}

// StorageRequestToProto maps a broker.StorageRequest to pb.StorageRequest.
func StorageRequestToProto(br broker.StorageRequest) (*pb.StorageRequest, error) {
	var pbStatus pb.StorageRequest_Status
	switch br.Status {
	case broker.RequestUnknown:
		pbStatus = pb.StorageRequest_UNSPECIFIED
	case broker.RequestBatching:
		pbStatus = pb.StorageRequest_BATCHING
	case broker.RequestPreparing:
		pbStatus = pb.StorageRequest_PREPARING
	case broker.RequestAuctioning:
		pbStatus = pb.StorageRequest_AUCTIONING
	case broker.RequestDealMaking:
		pbStatus = pb.StorageRequest_DEALMAKING
	case broker.RequestSuccess:
		pbStatus = pb.StorageRequest_SUCCESS
	case broker.RequestError:
		pbStatus = pb.StorageRequest_ERROR
	default:
		return nil, fmt.Errorf("unknown status: %d", br.Status)
	}

	return &pb.StorageRequest{
		Id:        string(br.ID),
		DataCid:   br.DataCid.Bytes(),
		Status:    pbStatus,
		BatchId:   string(br.BatchID),
		Origin:    br.Origin,
		CreatedAt: timestamppb.New(br.CreatedAt),
		UpdatedAt: timestamppb.New(br.UpdatedAt),
	}, nil
}

// StorageRequestInfoToProto maps a broker.
func StorageRequestInfoToProto(br broker.StorageRequestInfo) (*pb.GetStorageRequestInfoResponse, error) {
	protobr, err := StorageRequestToProto(br.StorageRequest)
	if err != nil {
		return nil, fmt.Errorf("creating proto for storage request: %s", err)
	}

	deals := make([]*pb.GetStorageRequestInfoResponse_StorageRequestDeal, len(br.Deals))
	for i, d := range br.Deals {
		deal := &pb.GetStorageRequestInfoResponse_StorageRequestDeal{
			StorageProviderId: d.StorageProviderID,
			DealId:            d.DealID,
			Expiration:        d.Expiration,
		}
		deals[i] = deal
	}

	return &pb.GetStorageRequestInfoResponse{
		StorageRequest: protobr,
		Deals:          deals,
	}, nil
}
