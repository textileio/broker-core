// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.15.2
// source: broker/v1/broker.proto

package broker

import (
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

type BrokerRequest_Status int32

const (
	BrokerRequest_UNSPECIFIED BrokerRequest_Status = 0
	BrokerRequest_BATCHING    BrokerRequest_Status = 1
	BrokerRequest_PREPARING   BrokerRequest_Status = 2
	BrokerRequest_AUCTIONING  BrokerRequest_Status = 3
	BrokerRequest_DEALMAKING  BrokerRequest_Status = 4
	BrokerRequest_SUCCESS     BrokerRequest_Status = 5
)

// Enum value maps for BrokerRequest_Status.
var (
	BrokerRequest_Status_name = map[int32]string{
		0: "UNSPECIFIED",
		1: "BATCHING",
		2: "PREPARING",
		3: "AUCTIONING",
		4: "DEALMAKING",
		5: "SUCCESS",
	}
	BrokerRequest_Status_value = map[string]int32{
		"UNSPECIFIED": 0,
		"BATCHING":    1,
		"PREPARING":   2,
		"AUCTIONING":  3,
		"DEALMAKING":  4,
		"SUCCESS":     5,
	}
)

func (x BrokerRequest_Status) Enum() *BrokerRequest_Status {
	p := new(BrokerRequest_Status)
	*p = x
	return p
}

func (x BrokerRequest_Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (BrokerRequest_Status) Descriptor() protoreflect.EnumDescriptor {
	return file_broker_v1_broker_proto_enumTypes[0].Descriptor()
}

func (BrokerRequest_Status) Type() protoreflect.EnumType {
	return &file_broker_v1_broker_proto_enumTypes[0]
}

func (x BrokerRequest_Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use BrokerRequest_Status.Descriptor instead.
func (BrokerRequest_Status) EnumDescriptor() ([]byte, []int) {
	return file_broker_v1_broker_proto_rawDescGZIP(), []int{6, 0}
}

type CreateBrokerRequestRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Cid  string                  `protobuf:"bytes,1,opt,name=cid,proto3" json:"cid,omitempty"`
	Meta *BrokerRequest_Metadata `protobuf:"bytes,2,opt,name=meta,proto3" json:"meta,omitempty"`
}

func (x *CreateBrokerRequestRequest) Reset() {
	*x = CreateBrokerRequestRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_v1_broker_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateBrokerRequestRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateBrokerRequestRequest) ProtoMessage() {}

func (x *CreateBrokerRequestRequest) ProtoReflect() protoreflect.Message {
	mi := &file_broker_v1_broker_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateBrokerRequestRequest.ProtoReflect.Descriptor instead.
func (*CreateBrokerRequestRequest) Descriptor() ([]byte, []int) {
	return file_broker_v1_broker_proto_rawDescGZIP(), []int{0}
}

func (x *CreateBrokerRequestRequest) GetCid() string {
	if x != nil {
		return x.Cid
	}
	return ""
}

func (x *CreateBrokerRequestRequest) GetMeta() *BrokerRequest_Metadata {
	if x != nil {
		return x.Meta
	}
	return nil
}

type CreateBrokerRequestResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Request *BrokerRequest `protobuf:"bytes,1,opt,name=request,proto3" json:"request,omitempty"`
}

func (x *CreateBrokerRequestResponse) Reset() {
	*x = CreateBrokerRequestResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_v1_broker_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateBrokerRequestResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateBrokerRequestResponse) ProtoMessage() {}

func (x *CreateBrokerRequestResponse) ProtoReflect() protoreflect.Message {
	mi := &file_broker_v1_broker_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateBrokerRequestResponse.ProtoReflect.Descriptor instead.
func (*CreateBrokerRequestResponse) Descriptor() ([]byte, []int) {
	return file_broker_v1_broker_proto_rawDescGZIP(), []int{1}
}

func (x *CreateBrokerRequestResponse) GetRequest() *BrokerRequest {
	if x != nil {
		return x.Request
	}
	return nil
}

type GetBrokerRequestRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *GetBrokerRequestRequest) Reset() {
	*x = GetBrokerRequestRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_v1_broker_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetBrokerRequestRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBrokerRequestRequest) ProtoMessage() {}

func (x *GetBrokerRequestRequest) ProtoReflect() protoreflect.Message {
	mi := &file_broker_v1_broker_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetBrokerRequestRequest.ProtoReflect.Descriptor instead.
func (*GetBrokerRequestRequest) Descriptor() ([]byte, []int) {
	return file_broker_v1_broker_proto_rawDescGZIP(), []int{2}
}

func (x *GetBrokerRequestRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type GetBrokerRequestResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BrokerRequest *BrokerRequest `protobuf:"bytes,1,opt,name=broker_request,json=brokerRequest,proto3" json:"broker_request,omitempty"`
}

func (x *GetBrokerRequestResponse) Reset() {
	*x = GetBrokerRequestResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_v1_broker_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetBrokerRequestResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBrokerRequestResponse) ProtoMessage() {}

func (x *GetBrokerRequestResponse) ProtoReflect() protoreflect.Message {
	mi := &file_broker_v1_broker_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetBrokerRequestResponse.ProtoReflect.Descriptor instead.
func (*GetBrokerRequestResponse) Descriptor() ([]byte, []int) {
	return file_broker_v1_broker_proto_rawDescGZIP(), []int{3}
}

func (x *GetBrokerRequestResponse) GetBrokerRequest() *BrokerRequest {
	if x != nil {
		return x.BrokerRequest
	}
	return nil
}

type CreateStorageDealRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BatchCid         string   `protobuf:"bytes,1,opt,name=batch_cid,json=batchCid,proto3" json:"batch_cid,omitempty"`
	BrokerRequestIds []string `protobuf:"bytes,2,rep,name=broker_request_ids,json=brokerRequestIds,proto3" json:"broker_request_ids,omitempty"`
}

func (x *CreateStorageDealRequest) Reset() {
	*x = CreateStorageDealRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_v1_broker_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateStorageDealRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateStorageDealRequest) ProtoMessage() {}

func (x *CreateStorageDealRequest) ProtoReflect() protoreflect.Message {
	mi := &file_broker_v1_broker_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateStorageDealRequest.ProtoReflect.Descriptor instead.
func (*CreateStorageDealRequest) Descriptor() ([]byte, []int) {
	return file_broker_v1_broker_proto_rawDescGZIP(), []int{4}
}

func (x *CreateStorageDealRequest) GetBatchCid() string {
	if x != nil {
		return x.BatchCid
	}
	return ""
}

func (x *CreateStorageDealRequest) GetBrokerRequestIds() []string {
	if x != nil {
		return x.BrokerRequestIds
	}
	return nil
}

type CreateStorageDealResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *CreateStorageDealResponse) Reset() {
	*x = CreateStorageDealResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_v1_broker_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateStorageDealResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateStorageDealResponse) ProtoMessage() {}

func (x *CreateStorageDealResponse) ProtoReflect() protoreflect.Message {
	mi := &file_broker_v1_broker_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateStorageDealResponse.ProtoReflect.Descriptor instead.
func (*CreateStorageDealResponse) Descriptor() ([]byte, []int) {
	return file_broker_v1_broker_proto_rawDescGZIP(), []int{5}
}

func (x *CreateStorageDealResponse) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

// General
type BrokerRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id            string                  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	DataCid       string                  `protobuf:"bytes,2,opt,name=data_cid,json=dataCid,proto3" json:"data_cid,omitempty"`
	Status        BrokerRequest_Status    `protobuf:"varint,3,opt,name=status,proto3,enum=broker.v1.BrokerRequest_Status" json:"status,omitempty"`
	Meta          *BrokerRequest_Metadata `protobuf:"bytes,4,opt,name=meta,proto3" json:"meta,omitempty"`
	StorageDealId string                  `protobuf:"bytes,5,opt,name=storage_deal_id,json=storageDealId,proto3" json:"storage_deal_id,omitempty"`
	CreatedAt     *timestamppb.Timestamp  `protobuf:"bytes,6,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt     *timestamppb.Timestamp  `protobuf:"bytes,7,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
}

func (x *BrokerRequest) Reset() {
	*x = BrokerRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_v1_broker_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BrokerRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BrokerRequest) ProtoMessage() {}

func (x *BrokerRequest) ProtoReflect() protoreflect.Message {
	mi := &file_broker_v1_broker_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BrokerRequest.ProtoReflect.Descriptor instead.
func (*BrokerRequest) Descriptor() ([]byte, []int) {
	return file_broker_v1_broker_proto_rawDescGZIP(), []int{6}
}

func (x *BrokerRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *BrokerRequest) GetDataCid() string {
	if x != nil {
		return x.DataCid
	}
	return ""
}

func (x *BrokerRequest) GetStatus() BrokerRequest_Status {
	if x != nil {
		return x.Status
	}
	return BrokerRequest_UNSPECIFIED
}

func (x *BrokerRequest) GetMeta() *BrokerRequest_Metadata {
	if x != nil {
		return x.Meta
	}
	return nil
}

func (x *BrokerRequest) GetStorageDealId() string {
	if x != nil {
		return x.StorageDealId
	}
	return ""
}

func (x *BrokerRequest) GetCreatedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.CreatedAt
	}
	return nil
}

func (x *BrokerRequest) GetUpdatedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.UpdatedAt
	}
	return nil
}

type BrokerRequest_Metadata struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Region string `protobuf:"bytes,1,opt,name=region,proto3" json:"region,omitempty"`
}

func (x *BrokerRequest_Metadata) Reset() {
	*x = BrokerRequest_Metadata{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_v1_broker_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BrokerRequest_Metadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BrokerRequest_Metadata) ProtoMessage() {}

func (x *BrokerRequest_Metadata) ProtoReflect() protoreflect.Message {
	mi := &file_broker_v1_broker_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BrokerRequest_Metadata.ProtoReflect.Descriptor instead.
func (*BrokerRequest_Metadata) Descriptor() ([]byte, []int) {
	return file_broker_v1_broker_proto_rawDescGZIP(), []int{6, 0}
}

func (x *BrokerRequest_Metadata) GetRegion() string {
	if x != nil {
		return x.Region
	}
	return ""
}

var File_broker_v1_broker_proto protoreflect.FileDescriptor

var file_broker_v1_broker_proto_rawDesc = []byte{
	0x0a, 0x16, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x62, 0x72, 0x6f, 0x6b,
	0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72,
	0x2e, 0x76, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x65, 0x0a, 0x1a, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x72,
	0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x63, 0x69, 0x64, 0x12, 0x35, 0x0a, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x21, 0x2e, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x42,
	0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x4d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x22, 0x51, 0x0a, 0x1b, 0x43,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x32, 0x0a, 0x07, 0x72, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x62, 0x72,
	0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x29,
	0x0a, 0x17, 0x47, 0x65, 0x74, 0x42, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x22, 0x5b, 0x0a, 0x18, 0x47, 0x65, 0x74,
	0x42, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3f, 0x0a, 0x0e, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x5f,
	0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e,
	0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x72, 0x6f, 0x6b, 0x65, 0x72,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x0d, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x65, 0x0a, 0x18, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x53, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x44, 0x65, 0x61, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x62, 0x61, 0x74, 0x63, 0x68, 0x5f, 0x63, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x62, 0x61, 0x74, 0x63, 0x68, 0x43, 0x69, 0x64, 0x12,
	0x2c, 0x0a, 0x12, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x5f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x5f, 0x69, 0x64, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x10, 0x62, 0x72, 0x6f,
	0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x49, 0x64, 0x73, 0x22, 0x2b, 0x0a,
	0x19, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x53, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x44, 0x65,
	0x61, 0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x22, 0xd1, 0x03, 0x0a, 0x0d, 0x42,
	0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x19, 0x0a, 0x08,
	0x64, 0x61, 0x74, 0x61, 0x5f, 0x63, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x64, 0x61, 0x74, 0x61, 0x43, 0x69, 0x64, 0x12, 0x37, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1f, 0x2e, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72,
	0x2e, 0x76, 0x31, 0x2e, 0x42, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x12, 0x35, 0x0a, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x21,
	0x2e, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x72, 0x6f, 0x6b, 0x65,
	0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0x52, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x12, 0x26, 0x0a, 0x0f, 0x73, 0x74, 0x6f, 0x72, 0x61,
	0x67, 0x65, 0x5f, 0x64, 0x65, 0x61, 0x6c, 0x5f, 0x69, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0d, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x44, 0x65, 0x61, 0x6c, 0x49, 0x64, 0x12,
	0x39, 0x0a, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52,
	0x09, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12, 0x39, 0x0a, 0x0a, 0x75, 0x70,
	0x64, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x75, 0x70, 0x64, 0x61,
	0x74, 0x65, 0x64, 0x41, 0x74, 0x1a, 0x22, 0x0a, 0x08, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e, 0x22, 0x63, 0x0a, 0x06, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x12, 0x0f, 0x0a, 0x0b, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49,
	0x45, 0x44, 0x10, 0x00, 0x12, 0x0c, 0x0a, 0x08, 0x42, 0x41, 0x54, 0x43, 0x48, 0x49, 0x4e, 0x47,
	0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x50, 0x52, 0x45, 0x50, 0x41, 0x52, 0x49, 0x4e, 0x47, 0x10,
	0x02, 0x12, 0x0e, 0x0a, 0x0a, 0x41, 0x55, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x49, 0x4e, 0x47, 0x10,
	0x03, 0x12, 0x0e, 0x0a, 0x0a, 0x44, 0x45, 0x41, 0x4c, 0x4d, 0x41, 0x4b, 0x49, 0x4e, 0x47, 0x10,
	0x04, 0x12, 0x0b, 0x0a, 0x07, 0x53, 0x55, 0x43, 0x43, 0x45, 0x53, 0x53, 0x10, 0x05, 0x32, 0xb5,
	0x02, 0x0a, 0x0a, 0x41, 0x50, 0x49, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x66, 0x0a,
	0x13, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x25, 0x2e, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x76, 0x31,
	0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x26, 0x2e, 0x62, 0x72,
	0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x72,
	0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x5d, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x42, 0x72, 0x6f, 0x6b,
	0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x22, 0x2e, 0x62, 0x72, 0x6f, 0x6b,
	0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x42, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x23, 0x2e,
	0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x42, 0x72, 0x6f,
	0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x00, 0x12, 0x60, 0x0a, 0x11, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x53, 0x74,
	0x6f, 0x72, 0x61, 0x67, 0x65, 0x44, 0x65, 0x61, 0x6c, 0x12, 0x23, 0x2e, 0x62, 0x72, 0x6f, 0x6b,
	0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x53, 0x74, 0x6f, 0x72,
	0x61, 0x67, 0x65, 0x44, 0x65, 0x61, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x24,
	0x2e, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74,
	0x65, 0x53, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x44, 0x65, 0x61, 0x6c, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x37, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x65, 0x78, 0x74, 0x69, 0x6c, 0x65, 0x69, 0x6f, 0x2f, 0x62,
	0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2d, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x62,
	0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x3b, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_broker_v1_broker_proto_rawDescOnce sync.Once
	file_broker_v1_broker_proto_rawDescData = file_broker_v1_broker_proto_rawDesc
)

func file_broker_v1_broker_proto_rawDescGZIP() []byte {
	file_broker_v1_broker_proto_rawDescOnce.Do(func() {
		file_broker_v1_broker_proto_rawDescData = protoimpl.X.CompressGZIP(file_broker_v1_broker_proto_rawDescData)
	})
	return file_broker_v1_broker_proto_rawDescData
}

var file_broker_v1_broker_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_broker_v1_broker_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_broker_v1_broker_proto_goTypes = []interface{}{
	(BrokerRequest_Status)(0),           // 0: broker.v1.BrokerRequest.Status
	(*CreateBrokerRequestRequest)(nil),  // 1: broker.v1.CreateBrokerRequestRequest
	(*CreateBrokerRequestResponse)(nil), // 2: broker.v1.CreateBrokerRequestResponse
	(*GetBrokerRequestRequest)(nil),     // 3: broker.v1.GetBrokerRequestRequest
	(*GetBrokerRequestResponse)(nil),    // 4: broker.v1.GetBrokerRequestResponse
	(*CreateStorageDealRequest)(nil),    // 5: broker.v1.CreateStorageDealRequest
	(*CreateStorageDealResponse)(nil),   // 6: broker.v1.CreateStorageDealResponse
	(*BrokerRequest)(nil),               // 7: broker.v1.BrokerRequest
	(*BrokerRequest_Metadata)(nil),      // 8: broker.v1.BrokerRequest.Metadata
	(*timestamppb.Timestamp)(nil),       // 9: google.protobuf.Timestamp
}
var file_broker_v1_broker_proto_depIdxs = []int32{
	8,  // 0: broker.v1.CreateBrokerRequestRequest.meta:type_name -> broker.v1.BrokerRequest.Metadata
	7,  // 1: broker.v1.CreateBrokerRequestResponse.request:type_name -> broker.v1.BrokerRequest
	7,  // 2: broker.v1.GetBrokerRequestResponse.broker_request:type_name -> broker.v1.BrokerRequest
	0,  // 3: broker.v1.BrokerRequest.status:type_name -> broker.v1.BrokerRequest.Status
	8,  // 4: broker.v1.BrokerRequest.meta:type_name -> broker.v1.BrokerRequest.Metadata
	9,  // 5: broker.v1.BrokerRequest.created_at:type_name -> google.protobuf.Timestamp
	9,  // 6: broker.v1.BrokerRequest.updated_at:type_name -> google.protobuf.Timestamp
	1,  // 7: broker.v1.APIService.CreateBrokerRequest:input_type -> broker.v1.CreateBrokerRequestRequest
	3,  // 8: broker.v1.APIService.GetBrokerRequest:input_type -> broker.v1.GetBrokerRequestRequest
	5,  // 9: broker.v1.APIService.CreateStorageDeal:input_type -> broker.v1.CreateStorageDealRequest
	2,  // 10: broker.v1.APIService.CreateBrokerRequest:output_type -> broker.v1.CreateBrokerRequestResponse
	4,  // 11: broker.v1.APIService.GetBrokerRequest:output_type -> broker.v1.GetBrokerRequestResponse
	6,  // 12: broker.v1.APIService.CreateStorageDeal:output_type -> broker.v1.CreateStorageDealResponse
	10, // [10:13] is the sub-list for method output_type
	7,  // [7:10] is the sub-list for method input_type
	7,  // [7:7] is the sub-list for extension type_name
	7,  // [7:7] is the sub-list for extension extendee
	0,  // [0:7] is the sub-list for field type_name
}

func init() { file_broker_v1_broker_proto_init() }
func file_broker_v1_broker_proto_init() {
	if File_broker_v1_broker_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_broker_v1_broker_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateBrokerRequestRequest); i {
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
		file_broker_v1_broker_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateBrokerRequestResponse); i {
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
		file_broker_v1_broker_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetBrokerRequestRequest); i {
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
		file_broker_v1_broker_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetBrokerRequestResponse); i {
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
		file_broker_v1_broker_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateStorageDealRequest); i {
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
		file_broker_v1_broker_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateStorageDealResponse); i {
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
		file_broker_v1_broker_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BrokerRequest); i {
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
		file_broker_v1_broker_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BrokerRequest_Metadata); i {
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
			RawDescriptor: file_broker_v1_broker_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_broker_v1_broker_proto_goTypes,
		DependencyIndexes: file_broker_v1_broker_proto_depIdxs,
		EnumInfos:         file_broker_v1_broker_proto_enumTypes,
		MessageInfos:      file_broker_v1_broker_proto_msgTypes,
	}.Build()
	File_broker_v1_broker_proto = out.File
	file_broker_v1_broker_proto_rawDesc = nil
	file_broker_v1_broker_proto_goTypes = nil
	file_broker_v1_broker_proto_depIdxs = nil
}
