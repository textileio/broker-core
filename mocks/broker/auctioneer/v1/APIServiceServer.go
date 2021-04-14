// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	context "context"

	auctioneer "github.com/textileio/broker-core/gen/broker/auctioneer/v1"

	mock "github.com/stretchr/testify/mock"
)

// APIServiceServer is an autogenerated mock type for the APIServiceServer type
type APIServiceServer struct {
	mock.Mock
}

// CreateAuction provides a mock function with given fields: _a0, _a1
func (_m *APIServiceServer) CreateAuction(_a0 context.Context, _a1 *auctioneer.CreateAuctionRequest) (*auctioneer.CreateAuctionResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *auctioneer.CreateAuctionResponse
	if rf, ok := ret.Get(0).(func(context.Context, *auctioneer.CreateAuctionRequest) *auctioneer.CreateAuctionResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*auctioneer.CreateAuctionResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *auctioneer.CreateAuctionRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAuction provides a mock function with given fields: _a0, _a1
func (_m *APIServiceServer) GetAuction(_a0 context.Context, _a1 *auctioneer.GetAuctionRequest) (*auctioneer.GetAuctionResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *auctioneer.GetAuctionResponse
	if rf, ok := ret.Get(0).(func(context.Context, *auctioneer.GetAuctionRequest) *auctioneer.GetAuctionResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*auctioneer.GetAuctionResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *auctioneer.GetAuctionRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mustEmbedUnimplementedAPIServiceServer provides a mock function with given fields:
func (_m *APIServiceServer) mustEmbedUnimplementedAPIServiceServer() {
	_m.Called()
}
