// Code generated by mockery v2.52.3. DO NOT EDIT.

package mocks

import (
	http "net/http"

	mock "github.com/stretchr/testify/mock"

	url "net/url"
)

// ProxyFunc is an autogenerated mock type for the ProxyFunc type
type ProxyFunc struct {
	mock.Mock
}

type ProxyFunc_Expecter struct {
	mock *mock.Mock
}

func (_m *ProxyFunc) EXPECT() *ProxyFunc_Expecter {
	return &ProxyFunc_Expecter{mock: &_m.Mock}
}

// Execute provides a mock function with given fields: _a0
func (_m *ProxyFunc) Execute(_a0 *http.Request) (*url.URL, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Execute")
	}

	var r0 *url.URL
	var r1 error
	if rf, ok := ret.Get(0).(func(*http.Request) (*url.URL, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(*http.Request) *url.URL); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*url.URL)
		}
	}

	if rf, ok := ret.Get(1).(func(*http.Request) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ProxyFunc_Execute_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Execute'
type ProxyFunc_Execute_Call struct {
	*mock.Call
}

// Execute is a helper method to define mock.On call
//   - _a0 *http.Request
func (_e *ProxyFunc_Expecter) Execute(_a0 interface{}) *ProxyFunc_Execute_Call {
	return &ProxyFunc_Execute_Call{Call: _e.mock.On("Execute", _a0)}
}

func (_c *ProxyFunc_Execute_Call) Run(run func(_a0 *http.Request)) *ProxyFunc_Execute_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*http.Request))
	})
	return _c
}

func (_c *ProxyFunc_Execute_Call) Return(_a0 *url.URL, _a1 error) *ProxyFunc_Execute_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ProxyFunc_Execute_Call) RunAndReturn(run func(*http.Request) (*url.URL, error)) *ProxyFunc_Execute_Call {
	_c.Call.Return(run)
	return _c
}

// NewProxyFunc creates a new instance of ProxyFunc. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewProxyFunc(t interface {
	mock.TestingT
	Cleanup(func())
}) *ProxyFunc {
	mock := &ProxyFunc{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
