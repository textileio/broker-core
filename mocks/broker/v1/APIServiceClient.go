// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	context "context"

	broker "github.com/textileio/broker-core/gen/broker/v1"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"
)

// APIServiceClient is an autogenerated mock type for the APIServiceClient type
type APIServiceClient struct {
	mock.Mock
}

// CreateBR provides a mock function with given fields: ctx, in, opts
func (_m *APIServiceClient) CreateBR(ctx context.Context, in *broker.CreateBRRequest, opts ...grpc.CallOption) (*broker.CreateBRResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *broker.CreateBRResponse
	if rf, ok := ret.Get(0).(func(context.Context, *broker.CreateBRRequest, ...grpc.CallOption) *broker.CreateBRResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*broker.CreateBRResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *broker.CreateBRRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBR provides a mock function with given fields: ctx, in, opts
func (_m *APIServiceClient) GetBR(ctx context.Context, in *broker.GetBRRequest, opts ...grpc.CallOption) (*broker.GetBRResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *broker.GetBRResponse
	if rf, ok := ret.Get(0).(func(context.Context, *broker.GetBRRequest, ...grpc.CallOption) *broker.GetBRResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*broker.GetBRResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *broker.GetBRRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
