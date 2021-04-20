// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	context "context"

	chainapi "github.com/textileio/broker-core/gen/broker/chainapi/v1"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"
)

// ChainApiServiceClient is an autogenerated mock type for the ChainApiServiceClient type
type ChainApiServiceClient struct {
	mock.Mock
}

// HasFunds provides a mock function with given fields: ctx, in, opts
func (_m *ChainApiServiceClient) HasFunds(ctx context.Context, in *chainapi.HasFundsRequest, opts ...grpc.CallOption) (*chainapi.HasFundsResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *chainapi.HasFundsResponse
	if rf, ok := ret.Get(0).(func(context.Context, *chainapi.HasFundsRequest, ...grpc.CallOption) *chainapi.HasFundsResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chainapi.HasFundsResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *chainapi.HasFundsRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LockInfo provides a mock function with given fields: ctx, in, opts
func (_m *ChainApiServiceClient) LockInfo(ctx context.Context, in *chainapi.LockInfoRequest, opts ...grpc.CallOption) (*chainapi.LockInfoResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *chainapi.LockInfoResponse
	if rf, ok := ret.Get(0).(func(context.Context, *chainapi.LockInfoRequest, ...grpc.CallOption) *chainapi.LockInfoResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chainapi.LockInfoResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *chainapi.LockInfoRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// State provides a mock function with given fields: ctx, in, opts
func (_m *ChainApiServiceClient) State(ctx context.Context, in *chainapi.StateRequest, opts ...grpc.CallOption) (*chainapi.StateResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *chainapi.StateResponse
	if rf, ok := ret.Get(0).(func(context.Context, *chainapi.StateRequest, ...grpc.CallOption) *chainapi.StateResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chainapi.StateResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *chainapi.StateRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
