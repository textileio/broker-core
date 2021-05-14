// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// Authorizer is an autogenerated mock type for the Authorizer type
type Authorizer struct {
	mock.Mock
}

// IsAuthorized provides a mock function with given fields: ctx, identity
func (_m *Authorizer) IsAuthorized(ctx context.Context, identity string) (bool, string, error) {
	ret := _m.Called(ctx, identity)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(ctx, identity)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(context.Context, string) string); ok {
		r1 = rf(ctx, identity)
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string) error); ok {
		r2 = rf(ctx, identity)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
