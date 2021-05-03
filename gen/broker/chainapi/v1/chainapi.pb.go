// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.15.2
// source: broker/chainapi/v1/chainapi.proto

package chainapi

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type DepositInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Amount     string `protobuf:"bytes,1,opt,name=amount,proto3" json:"amount,omitempty"`
	Sender     string `protobuf:"bytes,2,opt,name=sender,proto3" json:"sender,omitempty"`
	Expiration uint64 `protobuf:"varint,3,opt,name=expiration,proto3" json:"expiration,omitempty"`
}

func (x *DepositInfo) Reset() {
	*x = DepositInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DepositInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DepositInfo) ProtoMessage() {}

func (x *DepositInfo) ProtoReflect() protoreflect.Message {
	mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DepositInfo.ProtoReflect.Descriptor instead.
func (*DepositInfo) Descriptor() ([]byte, []int) {
	return file_broker_chainapi_v1_chainapi_proto_rawDescGZIP(), []int{0}
}

func (x *DepositInfo) GetAmount() string {
	if x != nil {
		return x.Amount
	}
	return ""
}

func (x *DepositInfo) GetSender() string {
	if x != nil {
		return x.Sender
	}
	return ""
}

func (x *DepositInfo) GetExpiration() uint64 {
	if x != nil {
		return x.Expiration
	}
	return 0
}

type LockInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AccountId string       `protobuf:"bytes,1,opt,name=account_id,json=accountId,proto3" json:"account_id,omitempty"`
	BrokerId  string       `protobuf:"bytes,2,opt,name=broker_id,json=brokerId,proto3" json:"broker_id,omitempty"`
	Deposit   *DepositInfo `protobuf:"bytes,3,opt,name=deposit,proto3" json:"deposit,omitempty"`
}

func (x *LockInfo) Reset() {
	*x = LockInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LockInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LockInfo) ProtoMessage() {}

func (x *LockInfo) ProtoReflect() protoreflect.Message {
	mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LockInfo.ProtoReflect.Descriptor instead.
func (*LockInfo) Descriptor() ([]byte, []int) {
	return file_broker_chainapi_v1_chainapi_proto_rawDescGZIP(), []int{1}
}

func (x *LockInfo) GetAccountId() string {
	if x != nil {
		return x.AccountId
	}
	return ""
}

func (x *LockInfo) GetBrokerId() string {
	if x != nil {
		return x.BrokerId
	}
	return ""
}

func (x *LockInfo) GetDeposit() *DepositInfo {
	if x != nil {
		return x.Deposit
	}
	return nil
}

type LockInfoRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AccountId   string `protobuf:"bytes,1,opt,name=account_id,json=accountId,proto3" json:"account_id,omitempty"`
	BlockHeight uint64 `protobuf:"varint,2,opt,name=block_height,json=blockHeight,proto3" json:"block_height,omitempty"`
}

func (x *LockInfoRequest) Reset() {
	*x = LockInfoRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LockInfoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LockInfoRequest) ProtoMessage() {}

func (x *LockInfoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LockInfoRequest.ProtoReflect.Descriptor instead.
func (*LockInfoRequest) Descriptor() ([]byte, []int) {
	return file_broker_chainapi_v1_chainapi_proto_rawDescGZIP(), []int{2}
}

func (x *LockInfoRequest) GetAccountId() string {
	if x != nil {
		return x.AccountId
	}
	return ""
}

func (x *LockInfoRequest) GetBlockHeight() uint64 {
	if x != nil {
		return x.BlockHeight
	}
	return 0
}

type LockInfoResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LockInfo *LockInfo `protobuf:"bytes,1,opt,name=lock_info,json=lockInfo,proto3" json:"lock_info,omitempty"`
}

func (x *LockInfoResponse) Reset() {
	*x = LockInfoResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LockInfoResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LockInfoResponse) ProtoMessage() {}

func (x *LockInfoResponse) ProtoReflect() protoreflect.Message {
	mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LockInfoResponse.ProtoReflect.Descriptor instead.
func (*LockInfoResponse) Descriptor() ([]byte, []int) {
	return file_broker_chainapi_v1_chainapi_proto_rawDescGZIP(), []int{3}
}

func (x *LockInfoResponse) GetLockInfo() *LockInfo {
	if x != nil {
		return x.LockInfo
	}
	return nil
}

type HasFundsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BrokerId  string `protobuf:"bytes,1,opt,name=broker_id,json=brokerId,proto3" json:"broker_id,omitempty"`
	AccountId string `protobuf:"bytes,2,opt,name=account_id,json=accountId,proto3" json:"account_id,omitempty"`
}

func (x *HasFundsRequest) Reset() {
	*x = HasFundsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HasFundsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HasFundsRequest) ProtoMessage() {}

func (x *HasFundsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HasFundsRequest.ProtoReflect.Descriptor instead.
func (*HasFundsRequest) Descriptor() ([]byte, []int) {
	return file_broker_chainapi_v1_chainapi_proto_rawDescGZIP(), []int{4}
}

func (x *HasFundsRequest) GetBrokerId() string {
	if x != nil {
		return x.BrokerId
	}
	return ""
}

func (x *HasFundsRequest) GetAccountId() string {
	if x != nil {
		return x.AccountId
	}
	return ""
}

type HasFundsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	HasFunds bool `protobuf:"varint,1,opt,name=has_funds,json=hasFunds,proto3" json:"has_funds,omitempty"`
}

func (x *HasFundsResponse) Reset() {
	*x = HasFundsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HasFundsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HasFundsResponse) ProtoMessage() {}

func (x *HasFundsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HasFundsResponse.ProtoReflect.Descriptor instead.
func (*HasFundsResponse) Descriptor() ([]byte, []int) {
	return file_broker_chainapi_v1_chainapi_proto_rawDescGZIP(), []int{5}
}

func (x *HasFundsResponse) GetHasFunds() bool {
	if x != nil {
		return x.HasFunds
	}
	return false
}

type StateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *StateRequest) Reset() {
	*x = StateRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StateRequest) ProtoMessage() {}

func (x *StateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StateRequest.ProtoReflect.Descriptor instead.
func (*StateRequest) Descriptor() ([]byte, []int) {
	return file_broker_chainapi_v1_chainapi_proto_rawDescGZIP(), []int{6}
}

type StateResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LockedFunds map[string]*LockInfo `protobuf:"bytes,1,rep,name=locked_funds,json=lockedFunds,proto3" json:"locked_funds,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	BlockHeight uint64               `protobuf:"varint,2,opt,name=block_height,json=blockHeight,proto3" json:"block_height,omitempty"`
	BlockHash   string               `protobuf:"bytes,3,opt,name=block_hash,json=blockHash,proto3" json:"block_hash,omitempty"`
}

func (x *StateResponse) Reset() {
	*x = StateResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StateResponse) ProtoMessage() {}

func (x *StateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StateResponse.ProtoReflect.Descriptor instead.
func (*StateResponse) Descriptor() ([]byte, []int) {
	return file_broker_chainapi_v1_chainapi_proto_rawDescGZIP(), []int{7}
}

func (x *StateResponse) GetLockedFunds() map[string]*LockInfo {
	if x != nil {
		return x.LockedFunds
	}
	return nil
}

func (x *StateResponse) GetBlockHeight() uint64 {
	if x != nil {
		return x.BlockHeight
	}
	return 0
}

func (x *StateResponse) GetBlockHash() string {
	if x != nil {
		return x.BlockHash
	}
	return ""
}

var File_broker_chainapi_v1_chainapi_proto protoreflect.FileDescriptor

var file_broker_chainapi_v1_chainapi_proto_rawDesc = []byte{
	0x0a, 0x21, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2f, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70,
	0x69, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31,
	0x22, 0x5d, 0x0a, 0x0b, 0x44, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x12,
	0x16, 0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x65, 0x6e, 0x64, 0x65,
	0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x65, 0x6e, 0x64, 0x65, 0x72, 0x12,
	0x1e, 0x0a, 0x0a, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x0a, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22,
	0x7a, 0x0a, 0x08, 0x4c, 0x6f, 0x63, 0x6b, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x1d, 0x0a, 0x0a, 0x61,
	0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x62, 0x72,
	0x6f, 0x6b, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x62,
	0x72, 0x6f, 0x6b, 0x65, 0x72, 0x49, 0x64, 0x12, 0x32, 0x0a, 0x07, 0x64, 0x65, 0x70, 0x6f, 0x73,
	0x69, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e,
	0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x49, 0x6e,
	0x66, 0x6f, 0x52, 0x07, 0x64, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x22, 0x53, 0x0a, 0x0f, 0x4c,
	0x6f, 0x63, 0x6b, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d,
	0x0a, 0x0a, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x21, 0x0a,
	0x0c, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74,
	0x22, 0x46, 0x0a, 0x10, 0x4c, 0x6f, 0x63, 0x6b, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x32, 0x0a, 0x09, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x69, 0x6e, 0x66,
	0x6f, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61,
	0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x4c, 0x6f, 0x63, 0x6b, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x08,
	0x6c, 0x6f, 0x63, 0x6b, 0x49, 0x6e, 0x66, 0x6f, 0x22, 0x4d, 0x0a, 0x0f, 0x48, 0x61, 0x73, 0x46,
	0x75, 0x6e, 0x64, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x62,
	0x72, 0x6f, 0x6b, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x63, 0x63, 0x6f,
	0x75, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61, 0x63,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x49, 0x64, 0x22, 0x2f, 0x0a, 0x10, 0x48, 0x61, 0x73, 0x46, 0x75,
	0x6e, 0x64, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x68,
	0x61, 0x73, 0x5f, 0x66, 0x75, 0x6e, 0x64, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08,
	0x68, 0x61, 0x73, 0x46, 0x75, 0x6e, 0x64, 0x73, 0x22, 0x0e, 0x0a, 0x0c, 0x53, 0x74, 0x61, 0x74,
	0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0xf8, 0x01, 0x0a, 0x0d, 0x53, 0x74, 0x61,
	0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x4e, 0x0a, 0x0c, 0x6c, 0x6f,
	0x63, 0x6b, 0x65, 0x64, 0x5f, 0x66, 0x75, 0x6e, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x2b, 0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x53,
	0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x4c, 0x6f, 0x63,
	0x6b, 0x65, 0x64, 0x46, 0x75, 0x6e, 0x64, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0b, 0x6c,
	0x6f, 0x63, 0x6b, 0x65, 0x64, 0x46, 0x75, 0x6e, 0x64, 0x73, 0x12, 0x21, 0x0a, 0x0c, 0x62, 0x6c,
	0x6f, 0x63, 0x6b, 0x5f, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x1d, 0x0a,
	0x0a, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x1a, 0x55, 0x0a, 0x10,
	0x4c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x46, 0x75, 0x6e, 0x64, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x2b, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x15, 0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e,
	0x4c, 0x6f, 0x63, 0x6b, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a,
	0x02, 0x38, 0x01, 0x32, 0xe9, 0x01, 0x0a, 0x0f, 0x43, 0x68, 0x61, 0x69, 0x6e, 0x41, 0x70, 0x69,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x49, 0x0a, 0x08, 0x4c, 0x6f, 0x63, 0x6b, 0x49,
	0x6e, 0x66, 0x6f, 0x12, 0x1c, 0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x76,
	0x31, 0x2e, 0x4c, 0x6f, 0x63, 0x6b, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x1d, 0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e,
	0x4c, 0x6f, 0x63, 0x6b, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x00, 0x12, 0x49, 0x0a, 0x08, 0x48, 0x61, 0x73, 0x46, 0x75, 0x6e, 0x64, 0x73, 0x12, 0x1c,
	0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x61, 0x73,
	0x46, 0x75, 0x6e, 0x64, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1d, 0x2e, 0x63,
	0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x61, 0x73, 0x46, 0x75,
	0x6e, 0x64, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x40, 0x0a,
	0x05, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x19, 0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70,
	0x69, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x1a, 0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e,
	0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42,
	0x42, 0x5a, 0x40, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x65,
	0x78, 0x74, 0x69, 0x6c, 0x65, 0x69, 0x6f, 0x2f, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2d, 0x63,
	0x6f, 0x72, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2f, 0x63,
	0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x3b, 0x63, 0x68, 0x61, 0x69, 0x6e,
	0x61, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_broker_chainapi_v1_chainapi_proto_rawDescOnce sync.Once
	file_broker_chainapi_v1_chainapi_proto_rawDescData = file_broker_chainapi_v1_chainapi_proto_rawDesc
)

func file_broker_chainapi_v1_chainapi_proto_rawDescGZIP() []byte {
	file_broker_chainapi_v1_chainapi_proto_rawDescOnce.Do(func() {
		file_broker_chainapi_v1_chainapi_proto_rawDescData = protoimpl.X.CompressGZIP(file_broker_chainapi_v1_chainapi_proto_rawDescData)
	})
	return file_broker_chainapi_v1_chainapi_proto_rawDescData
}

var file_broker_chainapi_v1_chainapi_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_broker_chainapi_v1_chainapi_proto_goTypes = []interface{}{
	(*DepositInfo)(nil),      // 0: chainapi.v1.DepositInfo
	(*LockInfo)(nil),         // 1: chainapi.v1.LockInfo
	(*LockInfoRequest)(nil),  // 2: chainapi.v1.LockInfoRequest
	(*LockInfoResponse)(nil), // 3: chainapi.v1.LockInfoResponse
	(*HasFundsRequest)(nil),  // 4: chainapi.v1.HasFundsRequest
	(*HasFundsResponse)(nil), // 5: chainapi.v1.HasFundsResponse
	(*StateRequest)(nil),     // 6: chainapi.v1.StateRequest
	(*StateResponse)(nil),    // 7: chainapi.v1.StateResponse
	nil,                      // 8: chainapi.v1.StateResponse.LockedFundsEntry
}
var file_broker_chainapi_v1_chainapi_proto_depIdxs = []int32{
	0, // 0: chainapi.v1.LockInfo.deposit:type_name -> chainapi.v1.DepositInfo
	1, // 1: chainapi.v1.LockInfoResponse.lock_info:type_name -> chainapi.v1.LockInfo
	8, // 2: chainapi.v1.StateResponse.locked_funds:type_name -> chainapi.v1.StateResponse.LockedFundsEntry
	1, // 3: chainapi.v1.StateResponse.LockedFundsEntry.value:type_name -> chainapi.v1.LockInfo
	2, // 4: chainapi.v1.ChainApiService.LockInfo:input_type -> chainapi.v1.LockInfoRequest
	4, // 5: chainapi.v1.ChainApiService.HasFunds:input_type -> chainapi.v1.HasFundsRequest
	6, // 6: chainapi.v1.ChainApiService.State:input_type -> chainapi.v1.StateRequest
	3, // 7: chainapi.v1.ChainApiService.LockInfo:output_type -> chainapi.v1.LockInfoResponse
	5, // 8: chainapi.v1.ChainApiService.HasFunds:output_type -> chainapi.v1.HasFundsResponse
	7, // 9: chainapi.v1.ChainApiService.State:output_type -> chainapi.v1.StateResponse
	7, // [7:10] is the sub-list for method output_type
	4, // [4:7] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_broker_chainapi_v1_chainapi_proto_init() }
func file_broker_chainapi_v1_chainapi_proto_init() {
	if File_broker_chainapi_v1_chainapi_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_broker_chainapi_v1_chainapi_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DepositInfo); i {
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
		file_broker_chainapi_v1_chainapi_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LockInfo); i {
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
		file_broker_chainapi_v1_chainapi_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LockInfoRequest); i {
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
		file_broker_chainapi_v1_chainapi_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LockInfoResponse); i {
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
		file_broker_chainapi_v1_chainapi_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HasFundsRequest); i {
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
		file_broker_chainapi_v1_chainapi_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HasFundsResponse); i {
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
		file_broker_chainapi_v1_chainapi_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StateRequest); i {
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
		file_broker_chainapi_v1_chainapi_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StateResponse); i {
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
			RawDescriptor: file_broker_chainapi_v1_chainapi_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_broker_chainapi_v1_chainapi_proto_goTypes,
		DependencyIndexes: file_broker_chainapi_v1_chainapi_proto_depIdxs,
		MessageInfos:      file_broker_chainapi_v1_chainapi_proto_msgTypes,
	}.Build()
	File_broker_chainapi_v1_chainapi_proto = out.File
	file_broker_chainapi_v1_chainapi_proto_rawDesc = nil
	file_broker_chainapi_v1_chainapi_proto_goTypes = nil
	file_broker_chainapi_v1_chainapi_proto_depIdxs = nil
}
