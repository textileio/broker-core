// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.15.2
// source: broker/packer/v1/packer.proto

package packer

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	_ "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ReadyToPackRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BrokerRequestId string `protobuf:"bytes,1,opt,name=broker_request_id,json=brokerRequestId,proto3" json:"broker_request_id,omitempty"`
	DataCid         string `protobuf:"bytes,2,opt,name=data_cid,json=dataCid,proto3" json:"data_cid,omitempty"`
}

func (x *ReadyToPackRequest) Reset() {
	*x = ReadyToPackRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_packer_v1_packer_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReadyToPackRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReadyToPackRequest) ProtoMessage() {}

func (x *ReadyToPackRequest) ProtoReflect() protoreflect.Message {
	mi := &file_broker_packer_v1_packer_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReadyToPackRequest.ProtoReflect.Descriptor instead.
func (*ReadyToPackRequest) Descriptor() ([]byte, []int) {
	return file_broker_packer_v1_packer_proto_rawDescGZIP(), []int{0}
}

func (x *ReadyToPackRequest) GetBrokerRequestId() string {
	if x != nil {
		return x.BrokerRequestId
	}
	return ""
}

func (x *ReadyToPackRequest) GetDataCid() string {
	if x != nil {
		return x.DataCid
	}
	return ""
}

type ReadyToPackResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ReadyToPackResponse) Reset() {
	*x = ReadyToPackResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_broker_packer_v1_packer_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReadyToPackResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReadyToPackResponse) ProtoMessage() {}

func (x *ReadyToPackResponse) ProtoReflect() protoreflect.Message {
	mi := &file_broker_packer_v1_packer_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReadyToPackResponse.ProtoReflect.Descriptor instead.
func (*ReadyToPackResponse) Descriptor() ([]byte, []int) {
	return file_broker_packer_v1_packer_proto_rawDescGZIP(), []int{1}
}

var File_broker_packer_v1_packer_proto protoreflect.FileDescriptor

var file_broker_packer_v1_packer_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2f, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x72, 0x2f,
	0x76, 0x31, 0x2f, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x10, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x72, 0x2e, 0x76,
	0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x5b, 0x0a, 0x12, 0x52, 0x65, 0x61, 0x64, 0x79, 0x54, 0x6f, 0x50, 0x61, 0x63,
	0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2a, 0x0a, 0x11, 0x62, 0x72, 0x6f, 0x6b,
	0x65, 0x72, 0x5f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0f, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x49, 0x64, 0x12, 0x19, 0x0a, 0x08, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x63, 0x69, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x64, 0x61, 0x74, 0x61, 0x43, 0x69, 0x64, 0x22,
	0x15, 0x0a, 0x13, 0x52, 0x65, 0x61, 0x64, 0x79, 0x54, 0x6f, 0x50, 0x61, 0x63, 0x6b, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0x6a, 0x0a, 0x0a, 0x41, 0x50, 0x49, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x12, 0x5c, 0x0a, 0x0b, 0x52, 0x65, 0x61, 0x64, 0x79, 0x54, 0x6f, 0x50,
	0x61, 0x63, 0x6b, 0x12, 0x24, 0x2e, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x70, 0x61, 0x63,
	0x6b, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x61, 0x64, 0x79, 0x54, 0x6f, 0x50, 0x61,
	0x63, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x25, 0x2e, 0x62, 0x72, 0x6f, 0x6b,
	0x65, 0x72, 0x2e, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x61,
	0x64, 0x79, 0x54, 0x6f, 0x50, 0x61, 0x63, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x00, 0x42, 0x3e, 0x5a, 0x3c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x74, 0x65, 0x78, 0x74, 0x69, 0x6c, 0x65, 0x69, 0x6f, 0x2f, 0x62, 0x72, 0x6f, 0x6b, 0x65,
	0x72, 0x2d, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x62, 0x72, 0x6f, 0x6b, 0x65,
	0x72, 0x2f, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x3b, 0x70, 0x61, 0x63, 0x6b,
	0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_broker_packer_v1_packer_proto_rawDescOnce sync.Once
	file_broker_packer_v1_packer_proto_rawDescData = file_broker_packer_v1_packer_proto_rawDesc
)

func file_broker_packer_v1_packer_proto_rawDescGZIP() []byte {
	file_broker_packer_v1_packer_proto_rawDescOnce.Do(func() {
		file_broker_packer_v1_packer_proto_rawDescData = protoimpl.X.CompressGZIP(file_broker_packer_v1_packer_proto_rawDescData)
	})
	return file_broker_packer_v1_packer_proto_rawDescData
}

var file_broker_packer_v1_packer_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_broker_packer_v1_packer_proto_goTypes = []interface{}{
	(*ReadyToPackRequest)(nil),  // 0: broker.packer.v1.ReadyToPackRequest
	(*ReadyToPackResponse)(nil), // 1: broker.packer.v1.ReadyToPackResponse
}
var file_broker_packer_v1_packer_proto_depIdxs = []int32{
	0, // 0: broker.packer.v1.APIService.ReadyToPack:input_type -> broker.packer.v1.ReadyToPackRequest
	1, // 1: broker.packer.v1.APIService.ReadyToPack:output_type -> broker.packer.v1.ReadyToPackResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_broker_packer_v1_packer_proto_init() }
func file_broker_packer_v1_packer_proto_init() {
	if File_broker_packer_v1_packer_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_broker_packer_v1_packer_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReadyToPackRequest); i {
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
		file_broker_packer_v1_packer_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReadyToPackResponse); i {
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
			RawDescriptor: file_broker_packer_v1_packer_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_broker_packer_v1_packer_proto_goTypes,
		DependencyIndexes: file_broker_packer_v1_packer_proto_depIdxs,
		MessageInfos:      file_broker_packer_v1_packer_proto_msgTypes,
	}.Build()
	File_broker_packer_v1_packer_proto = out.File
	file_broker_packer_v1_packer_proto_rawDesc = nil
	file_broker_packer_v1_packer_proto_goTypes = nil
	file_broker_packer_v1_packer_proto_depIdxs = nil
}
