// Code generated by mockery v2.52.3. DO NOT EDIT.

package mocks

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// WDSessionConfig is an autogenerated mock type for the WDSessionConfig type
type WDSessionConfig struct {
	mock.Mock
}

type WDSessionConfig_Expecter struct {
	mock *mock.Mock
}

func (_m *WDSessionConfig) EXPECT() *WDSessionConfig_Expecter {
	return &WDSessionConfig_Expecter{mock: &_m.Mock}
}

// CreateTimeout provides a mock function with no fields
func (_m *WDSessionConfig) CreateTimeout() time.Duration {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for CreateTimeout")
	}

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func() time.Duration); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	return r0
}

// WDSessionConfig_CreateTimeout_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateTimeout'
type WDSessionConfig_CreateTimeout_Call struct {
	*mock.Call
}

// CreateTimeout is a helper method to define mock.On call
func (_e *WDSessionConfig_Expecter) CreateTimeout() *WDSessionConfig_CreateTimeout_Call {
	return &WDSessionConfig_CreateTimeout_Call{Call: _e.mock.On("CreateTimeout")}
}

func (_c *WDSessionConfig_CreateTimeout_Call) Run(run func()) *WDSessionConfig_CreateTimeout_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *WDSessionConfig_CreateTimeout_Call) Return(_a0 time.Duration) *WDSessionConfig_CreateTimeout_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *WDSessionConfig_CreateTimeout_Call) RunAndReturn(run func() time.Duration) *WDSessionConfig_CreateTimeout_Call {
	_c.Call.Return(run)
	return _c
}

// ProxyDelete provides a mock function with no fields
func (_m *WDSessionConfig) ProxyDelete() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ProxyDelete")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// WDSessionConfig_ProxyDelete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ProxyDelete'
type WDSessionConfig_ProxyDelete_Call struct {
	*mock.Call
}

// ProxyDelete is a helper method to define mock.On call
func (_e *WDSessionConfig_Expecter) ProxyDelete() *WDSessionConfig_ProxyDelete_Call {
	return &WDSessionConfig_ProxyDelete_Call{Call: _e.mock.On("ProxyDelete")}
}

func (_c *WDSessionConfig_ProxyDelete_Call) Run(run func()) *WDSessionConfig_ProxyDelete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *WDSessionConfig_ProxyDelete_Call) Return(_a0 bool) *WDSessionConfig_ProxyDelete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *WDSessionConfig_ProxyDelete_Call) RunAndReturn(run func() bool) *WDSessionConfig_ProxyDelete_Call {
	_c.Call.Return(run)
	return _c
}

// NewWDSessionConfig creates a new instance of WDSessionConfig. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewWDSessionConfig(t interface {
	mock.TestingT
	Cleanup(func())
}) *WDSessionConfig {
	mock := &WDSessionConfig{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
