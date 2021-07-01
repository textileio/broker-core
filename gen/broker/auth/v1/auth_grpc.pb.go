// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package auth

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

// AuthAPIServiceClient is the client API for AuthAPIService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AuthAPIServiceClient interface {
	Auth(ctx context.Context, in *AuthRequest, opts ...grpc.CallOption) (*AuthResponse, error)
}

type authAPIServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAuthAPIServiceClient(cc grpc.ClientConnInterface) AuthAPIServiceClient {
	return &authAPIServiceClient{cc}
}

func (c *authAPIServiceClient) Auth(ctx context.Context, in *AuthRequest, opts ...grpc.CallOption) (*AuthResponse, error) {
	out := new(AuthResponse)
	err := c.cc.Invoke(ctx, "/broker.auth.v1.AuthAPIService/Auth", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AuthAPIServiceServer is the server API for AuthAPIService service.
// All implementations must embed UnimplementedAuthAPIServiceServer
// for forward compatibility
type AuthAPIServiceServer interface {
	Auth(context.Context, *AuthRequest) (*AuthResponse, error)
	mustEmbedUnimplementedAuthAPIServiceServer()
}

// UnimplementedAuthAPIServiceServer must be embedded to have forward compatible implementations.
type UnimplementedAuthAPIServiceServer struct {
}

func (UnimplementedAuthAPIServiceServer) Auth(context.Context, *AuthRequest) (*AuthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Auth not implemented")
}
func (UnimplementedAuthAPIServiceServer) mustEmbedUnimplementedAuthAPIServiceServer() {}

// UnsafeAuthAPIServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AuthAPIServiceServer will
// result in compilation errors.
type UnsafeAuthAPIServiceServer interface {
	mustEmbedUnimplementedAuthAPIServiceServer()
}

func RegisterAuthAPIServiceServer(s grpc.ServiceRegistrar, srv AuthAPIServiceServer) {
	s.RegisterService(&AuthAPIService_ServiceDesc, srv)
}

func _AuthAPIService_Auth_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AuthAPIServiceServer).Auth(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/broker.auth.v1.AuthAPIService/Auth",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AuthAPIServiceServer).Auth(ctx, req.(*AuthRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// AuthAPIService_ServiceDesc is the grpc.ServiceDesc for AuthAPIService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AuthAPIService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "broker.auth.v1.AuthAPIService",
	HandlerType: (*AuthAPIServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Auth",
			Handler:    _AuthAPIService_Auth_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "broker/auth/v1/auth.proto",
}
