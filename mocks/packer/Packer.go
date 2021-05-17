// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	cid "github.com/ipfs/go-cid"
	broker "github.com/textileio/broker-core/broker"

	context "context"

	mock "github.com/stretchr/testify/mock"
)

// Packer is an autogenerated mock type for the Packer type
type Packer struct {
	mock.Mock
}

// ReadyToPack provides a mock function with given fields: ctx, id, dataCid
func (_m *Packer) ReadyToPack(ctx context.Context, id broker.BrokerRequestID, dataCid cid.Cid) error {
	ret := _m.Called(ctx, id, dataCid)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, broker.BrokerRequestID, cid.Cid) error); ok {
		r0 = rf(ctx, id, dataCid)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}