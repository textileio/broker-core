// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	context "context"

	broker "github.com/textileio/broker-core/broker"

	mock "github.com/stretchr/testify/mock"
)

// Auctioneer is an autogenerated mock type for the Auctioneer type
type Auctioneer struct {
	mock.Mock
}

// GetAuction provides a mock function with given fields: ctx, id
func (_m *Auctioneer) GetAuction(ctx context.Context, id broker.AuctionID) (broker.Auction, error) {
	ret := _m.Called(ctx, id)

	var r0 broker.Auction
	if rf, ok := ret.Get(0).(func(context.Context, broker.AuctionID) broker.Auction); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Get(0).(broker.Auction)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, broker.AuctionID) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReadyToAuction provides a mock function with given fields: ctx, id, dealSize, dealDuration
func (_m *Auctioneer) ReadyToAuction(ctx context.Context, id broker.StorageDealID, dealSize uint64, dealDuration uint64) (broker.AuctionID, error) {
	ret := _m.Called(ctx, id, dealSize, dealDuration)

	var r0 broker.AuctionID
	if rf, ok := ret.Get(0).(func(context.Context, broker.StorageDealID, uint64, uint64) broker.AuctionID); ok {
		r0 = rf(ctx, id, dealSize, dealDuration)
	} else {
		r0 = ret.Get(0).(broker.AuctionID)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, broker.StorageDealID, uint64, uint64) error); ok {
		r1 = rf(ctx, id, dealSize, dealDuration)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
