// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package chainapi

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ChainApiServiceClient is the client API for ChainApiService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ChainApiServiceClient interface {
	HasDeposit(ctx context.Context, in *HasDepositRequest, opts ...grpc.CallOption) (*HasDepositResponse, error)
}

type chainApiServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewChainApiServiceClient(cc grpc.ClientConnInterface) ChainApiServiceClient {
	return &chainApiServiceClient{cc}
}

func (c *chainApiServiceClient) HasDeposit(ctx context.Context, in *HasDepositRequest, opts ...grpc.CallOption) (*HasDepositResponse, error) {
	out := new(HasDepositResponse)
	err := c.cc.Invoke(ctx, "/chainapi.v1.ChainApiService/HasDeposit", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ChainApiServiceServer is the server API for ChainApiService service.
// All implementations must embed UnimplementedChainApiServiceServer
// for forward compatibility
type ChainApiServiceServer interface {
	HasDeposit(context.Context, *HasDepositRequest) (*HasDepositResponse, error)
	mustEmbedUnimplementedChainApiServiceServer()
}

// UnimplementedChainApiServiceServer must be embedded to have forward compatible implementations.
type UnimplementedChainApiServiceServer struct {
}

func (UnimplementedChainApiServiceServer) HasDeposit(context.Context, *HasDepositRequest) (*HasDepositResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HasDeposit not implemented")
}
func (UnimplementedChainApiServiceServer) mustEmbedUnimplementedChainApiServiceServer() {}

// UnsafeChainApiServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ChainApiServiceServer will
// result in compilation errors.
type UnsafeChainApiServiceServer interface {
	mustEmbedUnimplementedChainApiServiceServer()
}

func RegisterChainApiServiceServer(s grpc.ServiceRegistrar, srv ChainApiServiceServer) {
	s.RegisterService(&ChainApiService_ServiceDesc, srv)
}

func _ChainApiService_HasDeposit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HasDepositRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChainApiServiceServer).HasDeposit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chainapi.v1.ChainApiService/HasDeposit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChainApiServiceServer).HasDeposit(ctx, req.(*HasDepositRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ChainApiService_ServiceDesc is the grpc.ServiceDesc for ChainApiService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ChainApiService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "chainapi.v1.ChainApiService",
	HandlerType: (*ChainApiServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "HasDeposit",
			Handler:    _ChainApiService_HasDeposit_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "broker/chainapi/v1/chainapi.proto",
}
