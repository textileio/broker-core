// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.15.2
// source: broker/auctioneer/v1/message/message.proto

package message

import (
	v1 "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Auction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id           string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	PayloadCid   string                 `protobuf:"bytes,2,opt,name=payload_cid,json=payloadCid,proto3" json:"payload_cid,omitempty"`
	DataUri      string                 `protobuf:"bytes,3,opt,name=data_uri,json=dataUri,proto3" json:"data_uri,omitempty"`
	DealSize     uint64                 `protobuf:"varint,4,opt,name=deal_size,json=dealSize,proto3" json:"deal_size,omitempty"`
	DealDuration uint64                 `protobuf:"varint,5,opt,name=deal_duration,json=dealDuration,proto3" json:"deal_duration,omitempty"`
	Sources      *v1.Sources            `protobuf:"bytes,6,opt,name=sources,proto3" json:"sources,omitempty"`
	EndsAt       *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=ends_at,json=endsAt,proto3" json:"ends_at,omitempty"`
}

func (x *Auction) Reset() {
	*x = Auction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_auctioneer_v1_message_message_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Auction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Auction) ProtoMessage() {}

func (x *Auction) ProtoReflect() protoreflect.Message {
	mi := &file_broker_auctioneer_v1_message_message_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Auction.ProtoReflect.Descriptor instead.
func (*Auction) Descriptor() ([]byte, []int) {
	return file_broker_auctioneer_v1_message_message_proto_rawDescGZIP(), []int{0}
}

func (x *Auction) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Auction) GetPayloadCid() string {
	if x != nil {
		return x.PayloadCid
	}
	return ""
}

func (x *Auction) GetDataUri() string {
	if x != nil {
		return x.DataUri
	}
	return ""
}

func (x *Auction) GetDealSize() uint64 {
	if x != nil {
		return x.DealSize
	}
	return 0
}

func (x *Auction) GetDealDuration() uint64 {
	if x != nil {
		return x.DealDuration
	}
	return 0
}

func (x *Auction) GetSources() *v1.Sources {
	if x != nil {
		return x.Sources
	}
	return nil
}

func (x *Auction) GetEndsAt() *timestamppb.Timestamp {
	if x != nil {
		return x.EndsAt
	}
	return nil
}

type Bid struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AuctionId        string `protobuf:"bytes,1,opt,name=auction_id,json=auctionId,proto3" json:"auction_id,omitempty"`
	MinerAddr        string `protobuf:"bytes,2,opt,name=miner_addr,json=minerAddr,proto3" json:"miner_addr,omitempty"`
	WalletAddrSig    []byte `protobuf:"bytes,3,opt,name=wallet_addr_sig,json=walletAddrSig,proto3" json:"wallet_addr_sig,omitempty"`
	AskPrice         int64  `protobuf:"varint,4,opt,name=ask_price,json=askPrice,proto3" json:"ask_price,omitempty"`
	VerifiedAskPrice int64  `protobuf:"varint,5,opt,name=verified_ask_price,json=verifiedAskPrice,proto3" json:"verified_ask_price,omitempty"`
	StartEpoch       uint64 `protobuf:"varint,6,opt,name=start_epoch,json=startEpoch,proto3" json:"start_epoch,omitempty"`
	FastRetrieval    bool   `protobuf:"varint,7,opt,name=fast_retrieval,json=fastRetrieval,proto3" json:"fast_retrieval,omitempty"`
}

func (x *Bid) Reset() {
	*x = Bid{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_auctioneer_v1_message_message_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Bid) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Bid) ProtoMessage() {}

func (x *Bid) ProtoReflect() protoreflect.Message {
	mi := &file_broker_auctioneer_v1_message_message_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Bid.ProtoReflect.Descriptor instead.
func (*Bid) Descriptor() ([]byte, []int) {
	return file_broker_auctioneer_v1_message_message_proto_rawDescGZIP(), []int{1}
}

func (x *Bid) GetAuctionId() string {
	if x != nil {
		return x.AuctionId
	}
	return ""
}

func (x *Bid) GetMinerAddr() string {
	if x != nil {
		return x.MinerAddr
	}
	return ""
}

func (x *Bid) GetWalletAddrSig() []byte {
	if x != nil {
		return x.WalletAddrSig
	}
	return nil
}

func (x *Bid) GetAskPrice() int64 {
	if x != nil {
		return x.AskPrice
	}
	return 0
}

func (x *Bid) GetVerifiedAskPrice() int64 {
	if x != nil {
		return x.VerifiedAskPrice
	}
	return 0
}

func (x *Bid) GetStartEpoch() uint64 {
	if x != nil {
		return x.StartEpoch
	}
	return 0
}

func (x *Bid) GetFastRetrieval() bool {
	if x != nil {
		return x.FastRetrieval
	}
	return false
}

type WinningBid struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AuctionId string `protobuf:"bytes,1,opt,name=auction_id,json=auctionId,proto3" json:"auction_id,omitempty"`
	BidId     string `protobuf:"bytes,2,opt,name=bid_id,json=bidId,proto3" json:"bid_id,omitempty"`
}

func (x *WinningBid) Reset() {
	*x = WinningBid{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_auctioneer_v1_message_message_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WinningBid) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WinningBid) ProtoMessage() {}

func (x *WinningBid) ProtoReflect() protoreflect.Message {
	mi := &file_broker_auctioneer_v1_message_message_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WinningBid.ProtoReflect.Descriptor instead.
func (*WinningBid) Descriptor() ([]byte, []int) {
	return file_broker_auctioneer_v1_message_message_proto_rawDescGZIP(), []int{2}
}

func (x *WinningBid) GetAuctionId() string {
	if x != nil {
		return x.AuctionId
	}
	return ""
}

func (x *WinningBid) GetBidId() string {
	if x != nil {
		return x.BidId
	}
	return ""
}

type WinningBidProposal struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AuctionId   string `protobuf:"bytes,1,opt,name=auction_id,json=auctionId,proto3" json:"auction_id,omitempty"`
	BidId       string `protobuf:"bytes,2,opt,name=bid_id,json=bidId,proto3" json:"bid_id,omitempty"`
	ProposalCid string `protobuf:"bytes,3,opt,name=proposal_cid,json=proposalCid,proto3" json:"proposal_cid,omitempty"`
}

func (x *WinningBidProposal) Reset() {
	*x = WinningBidProposal{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_auctioneer_v1_message_message_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WinningBidProposal) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WinningBidProposal) ProtoMessage() {}

func (x *WinningBidProposal) ProtoReflect() protoreflect.Message {
	mi := &file_broker_auctioneer_v1_message_message_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WinningBidProposal.ProtoReflect.Descriptor instead.
func (*WinningBidProposal) Descriptor() ([]byte, []int) {
	return file_broker_auctioneer_v1_message_message_proto_rawDescGZIP(), []int{3}
}

func (x *WinningBidProposal) GetAuctionId() string {
	if x != nil {
		return x.AuctionId
	}
	return ""
}

func (x *WinningBidProposal) GetBidId() string {
	if x != nil {
		return x.BidId
	}
	return ""
}

func (x *WinningBidProposal) GetProposalCid() string {
	if x != nil {
		return x.ProposalCid
	}
	return ""
}

var File_broker_auctioneer_v1_message_message_proto protoreflect.FileDescriptor

var file_broker_auctioneer_v1_message_message_proto_rawDesc = []byte{
	0x0a, 0x2a, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2f, 0x61, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x65, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2f, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1c, 0x62, 0x72,
	0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x61, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x65, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x25, 0x62, 0x72, 0x6f,
	0x6b, 0x65, 0x72, 0x2f, 0x61, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x65, 0x65, 0x72, 0x2f, 0x76,
	0x31, 0x2f, 0x61, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x65, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x85, 0x02, 0x0a, 0x07, 0x41, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x0e,
	0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1f,
	0x0a, 0x0b, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x5f, 0x63, 0x69, 0x64, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0a, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x43, 0x69, 0x64, 0x12,
	0x19, 0x0a, 0x08, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x75, 0x72, 0x69, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x64, 0x61, 0x74, 0x61, 0x55, 0x72, 0x69, 0x12, 0x1b, 0x0a, 0x09, 0x64, 0x65,
	0x61, 0x6c, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08, 0x64,
	0x65, 0x61, 0x6c, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x64, 0x65, 0x61, 0x6c, 0x5f,
	0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0c,
	0x64, 0x65, 0x61, 0x6c, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x37, 0x0a, 0x07,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e,
	0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x61, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x65, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x52, 0x07, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x73, 0x12, 0x33, 0x0a, 0x07, 0x65, 0x6e, 0x64, 0x73, 0x5f, 0x61, 0x74,
	0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x52, 0x06, 0x65, 0x6e, 0x64, 0x73, 0x41, 0x74, 0x22, 0xfe, 0x01, 0x0a, 0x03, 0x42,
	0x69, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49,
	0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x69, 0x6e, 0x65, 0x72, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6d, 0x69, 0x6e, 0x65, 0x72, 0x41, 0x64, 0x64, 0x72,
	0x12, 0x26, 0x0a, 0x0f, 0x77, 0x61, 0x6c, 0x6c, 0x65, 0x74, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x5f,
	0x73, 0x69, 0x67, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0d, 0x77, 0x61, 0x6c, 0x6c, 0x65,
	0x74, 0x41, 0x64, 0x64, 0x72, 0x53, 0x69, 0x67, 0x12, 0x1b, 0x0a, 0x09, 0x61, 0x73, 0x6b, 0x5f,
	0x70, 0x72, 0x69, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x61, 0x73, 0x6b,
	0x50, 0x72, 0x69, 0x63, 0x65, 0x12, 0x2c, 0x0a, 0x12, 0x76, 0x65, 0x72, 0x69, 0x66, 0x69, 0x65,
	0x64, 0x5f, 0x61, 0x73, 0x6b, 0x5f, 0x70, 0x72, 0x69, 0x63, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x10, 0x76, 0x65, 0x72, 0x69, 0x66, 0x69, 0x65, 0x64, 0x41, 0x73, 0x6b, 0x50, 0x72,
	0x69, 0x63, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x74, 0x61, 0x72, 0x74, 0x5f, 0x65, 0x70, 0x6f,
	0x63, 0x68, 0x18, 0x06, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0a, 0x73, 0x74, 0x61, 0x72, 0x74, 0x45,
	0x70, 0x6f, 0x63, 0x68, 0x12, 0x25, 0x0a, 0x0e, 0x66, 0x61, 0x73, 0x74, 0x5f, 0x72, 0x65, 0x74,
	0x72, 0x69, 0x65, 0x76, 0x61, 0x6c, 0x18, 0x07, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0d, 0x66, 0x61,
	0x73, 0x74, 0x52, 0x65, 0x74, 0x72, 0x69, 0x65, 0x76, 0x61, 0x6c, 0x22, 0x42, 0x0a, 0x0a, 0x57,
	0x69, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x42, 0x69, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x75, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61,
	0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x15, 0x0a, 0x06, 0x62, 0x69, 0x64, 0x5f,
	0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x62, 0x69, 0x64, 0x49, 0x64, 0x22,
	0x6d, 0x0a, 0x12, 0x57, 0x69, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x42, 0x69, 0x64, 0x50, 0x72, 0x6f,
	0x70, 0x6f, 0x73, 0x61, 0x6c, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61, 0x75, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x49, 0x64, 0x12, 0x15, 0x0a, 0x06, 0x62, 0x69, 0x64, 0x5f, 0x69, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x62, 0x69, 0x64, 0x49, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x70,
	0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x5f, 0x63, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0b, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x43, 0x69, 0x64, 0x42, 0x43,
	0x5a, 0x41, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x65, 0x78,
	0x74, 0x69, 0x6c, 0x65, 0x69, 0x6f, 0x2f, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2d, 0x63, 0x6f,
	0x72, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2f, 0x61, 0x75,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x65, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_broker_auctioneer_v1_message_message_proto_rawDescOnce sync.Once
	file_broker_auctioneer_v1_message_message_proto_rawDescData = file_broker_auctioneer_v1_message_message_proto_rawDesc
)

func file_broker_auctioneer_v1_message_message_proto_rawDescGZIP() []byte {
	file_broker_auctioneer_v1_message_message_proto_rawDescOnce.Do(func() {
		file_broker_auctioneer_v1_message_message_proto_rawDescData = protoimpl.X.CompressGZIP(file_broker_auctioneer_v1_message_message_proto_rawDescData)
	})
	return file_broker_auctioneer_v1_message_message_proto_rawDescData
}

var file_broker_auctioneer_v1_message_message_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_broker_auctioneer_v1_message_message_proto_goTypes = []interface{}{
	(*Auction)(nil),               // 0: broker.auctioneer.v1.message.Auction
	(*Bid)(nil),                   // 1: broker.auctioneer.v1.message.Bid
	(*WinningBid)(nil),            // 2: broker.auctioneer.v1.message.WinningBid
	(*WinningBidProposal)(nil),    // 3: broker.auctioneer.v1.message.WinningBidProposal
	(*v1.Sources)(nil),            // 4: broker.auctioneer.v1.Sources
	(*timestamppb.Timestamp)(nil), // 5: google.protobuf.Timestamp
}
var file_broker_auctioneer_v1_message_message_proto_depIdxs = []int32{
	4, // 0: broker.auctioneer.v1.message.Auction.sources:type_name -> broker.auctioneer.v1.Sources
	5, // 1: broker.auctioneer.v1.message.Auction.ends_at:type_name -> google.protobuf.Timestamp
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_broker_auctioneer_v1_message_message_proto_init() }
func file_broker_auctioneer_v1_message_message_proto_init() {
	if File_broker_auctioneer_v1_message_message_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_broker_auctioneer_v1_message_message_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Auction); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_broker_auctioneer_v1_message_message_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Bid); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_broker_auctioneer_v1_message_message_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WinningBid); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_broker_auctioneer_v1_message_message_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WinningBidProposal); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_broker_auctioneer_v1_message_message_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_broker_auctioneer_v1_message_message_proto_goTypes,
		DependencyIndexes: file_broker_auctioneer_v1_message_message_proto_depIdxs,
		MessageInfos:      file_broker_auctioneer_v1_message_message_proto_msgTypes,
	}.Build()
	File_broker_auctioneer_v1_message_message_proto = out.File
	file_broker_auctioneer_v1_message_message_proto_rawDesc = nil
	file_broker_auctioneer_v1_message_message_proto_goTypes = nil
	file_broker_auctioneer_v1_message_message_proto_depIdxs = nil
}
