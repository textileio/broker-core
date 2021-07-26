// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	cid "github.com/ipfs/go-cid"
	broker "github.com/textileio/broker-core/broker"

	context "context"

	mock "github.com/stretchr/testify/mock"
)

// Broker is an autogenerated mock type for the Broker type
type Broker struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, dataCid
func (_m *Broker) Create(ctx context.Context, dataCid cid.Cid) (broker.StorageRequest, error) {
	ret := _m.Called(ctx, dataCid)

	var r0 broker.StorageRequest
	if rf, ok := ret.Get(0).(func(context.Context, cid.Cid) broker.StorageRequest); ok {
		r0 = rf(ctx, dataCid)
	} else {
		r0 = ret.Get(0).(broker.StorageRequest)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, cid.Cid) error); ok {
		r1 = rf(ctx, dataCid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreatePrepared provides a mock function with given fields: ctx, payloadCid, pc
func (_m *Broker) CreatePrepared(ctx context.Context, payloadCid cid.Cid, pc broker.PreparedCAR) (broker.StorageRequest, error) {
	ret := _m.Called(ctx, payloadCid, pc)

	var r0 broker.StorageRequest
	if rf, ok := ret.Get(0).(func(context.Context, cid.Cid, broker.PreparedCAR) broker.StorageRequest); ok {
		r0 = rf(ctx, payloadCid, pc)
	} else {
		r0 = ret.Get(0).(broker.StorageRequest)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, cid.Cid, broker.PreparedCAR) error); ok {
		r1 = rf(ctx, payloadCid, pc)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStorageRequestInfo provides a mock function with given fields: ctx, ID
func (_m *Broker) GetStorageRequestInfo(ctx context.Context, ID broker.StorageRequestID) (broker.StorageRequestInfo, error) {
	ret := _m.Called(ctx, ID)

	var r0 broker.StorageRequestInfo
	if rf, ok := ret.Get(0).(func(context.Context, broker.StorageRequestID) broker.StorageRequestInfo); ok {
		r0 = rf(ctx, ID)
	} else {
		r0 = ret.Get(0).(broker.StorageRequestInfo)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, broker.StorageRequestID) error); ok {
		r1 = rf(ctx, ID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
