// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
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

type HasDepositRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BrokerId  string `protobuf:"bytes,1,opt,name=broker_id,json=brokerId,proto3" json:"broker_id,omitempty"`
	AccountId string `protobuf:"bytes,2,opt,name=account_id,json=accountId,proto3" json:"account_id,omitempty"`
}

func (x *HasDepositRequest) Reset() {
	*x = HasDepositRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HasDepositRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HasDepositRequest) ProtoMessage() {}

func (x *HasDepositRequest) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use HasDepositRequest.ProtoReflect.Descriptor instead.
func (*HasDepositRequest) Descriptor() ([]byte, []int) {
	return file_broker_chainapi_v1_chainapi_proto_rawDescGZIP(), []int{0}
}

func (x *HasDepositRequest) GetBrokerId() string {
	if x != nil {
		return x.BrokerId
	}
	return ""
}

func (x *HasDepositRequest) GetAccountId() string {
	if x != nil {
		return x.AccountId
	}
	return ""
}

type HasDepositResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	HasDeposit bool `protobuf:"varint,1,opt,name=has_deposit,json=hasDeposit,proto3" json:"has_deposit,omitempty"`
}

func (x *HasDepositResponse) Reset() {
	*x = HasDepositResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_chainapi_v1_chainapi_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HasDepositResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HasDepositResponse) ProtoMessage() {}

func (x *HasDepositResponse) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use HasDepositResponse.ProtoReflect.Descriptor instead.
func (*HasDepositResponse) Descriptor() ([]byte, []int) {
	return file_broker_chainapi_v1_chainapi_proto_rawDescGZIP(), []int{1}
}

func (x *HasDepositResponse) GetHasDeposit() bool {
	if x != nil {
		return x.HasDeposit
	}
	return false
}

var File_broker_chainapi_v1_chainapi_proto protoreflect.FileDescriptor

var file_broker_chainapi_v1_chainapi_proto_rawDesc = []byte{
	0x0a, 0x21, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2f, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70,
	0x69, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x12, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x63, 0x68, 0x61, 0x69,
	0x6e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x22, 0x4f, 0x0a, 0x11, 0x48, 0x61, 0x73, 0x44, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1b, 0x0a, 0x09,
	0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x63, 0x63,
	0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61,
	0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x49, 0x64, 0x22, 0x35, 0x0a, 0x12, 0x48, 0x61, 0x73, 0x44,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1f,
	0x0a, 0x0b, 0x68, 0x61, 0x73, 0x5f, 0x64, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x0a, 0x68, 0x61, 0x73, 0x44, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x32,
	0x70, 0x0a, 0x0f, 0x43, 0x68, 0x61, 0x69, 0x6e, 0x41, 0x70, 0x69, 0x53, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x12, 0x5d, 0x0a, 0x0a, 0x48, 0x61, 0x73, 0x44, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x12, 0x25, 0x2e, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61,
	0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x61, 0x73, 0x44, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x26, 0x2e, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72,
	0x2e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x61, 0x73,
	0x44, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x42, 0x42, 0x5a, 0x40, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x74, 0x65, 0x78, 0x74, 0x69, 0x6c, 0x65, 0x69, 0x6f, 0x2f, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72,
	0x2d, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72,
	0x2f, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x3b, 0x63, 0x68, 0x61,
	0x69, 0x6e, 0x61, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
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

var file_broker_chainapi_v1_chainapi_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_broker_chainapi_v1_chainapi_proto_goTypes = []interface{}{
	(*HasDepositRequest)(nil),  // 0: broker.chainapi.v1.HasDepositRequest
	(*HasDepositResponse)(nil), // 1: broker.chainapi.v1.HasDepositResponse
}
var file_broker_chainapi_v1_chainapi_proto_depIdxs = []int32{
	0, // 0: broker.chainapi.v1.ChainApiService.HasDeposit:input_type -> broker.chainapi.v1.HasDepositRequest
	1, // 1: broker.chainapi.v1.ChainApiService.HasDeposit:output_type -> broker.chainapi.v1.HasDepositResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_broker_chainapi_v1_chainapi_proto_init() }
func file_broker_chainapi_v1_chainapi_proto_init() {
	if File_broker_chainapi_v1_chainapi_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_broker_chainapi_v1_chainapi_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HasDepositRequest); i {
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
			switch v := v.(*HasDepositResponse); i {
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
			NumMessages:   2,
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
