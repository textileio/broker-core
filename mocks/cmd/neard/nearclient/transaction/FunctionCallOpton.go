// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	transaction "github.com/textileio/broker-core/cmd/neard/nearclient/transaction"
)

// FunctionCallOpton is an autogenerated mock type for the FunctionCallOpton type
type FunctionCallOpton struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0
func (_m *FunctionCallOpton) Execute(_a0 *transaction.FunctionCall) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*transaction.FunctionCall) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
