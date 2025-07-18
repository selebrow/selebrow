// Code generated by mockery v2.52.3. DO NOT EDIT.

package mocks

import (
	capabilities "github.com/selebrow/selebrow/pkg/capabilities"
	mock "github.com/stretchr/testify/mock"
)

// GetHashFunc is an autogenerated mock type for the GetHashFunc type
type GetHashFunc struct {
	mock.Mock
}

type GetHashFunc_Expecter struct {
	mock *mock.Mock
}

func (_m *GetHashFunc) EXPECT() *GetHashFunc_Expecter {
	return &GetHashFunc_Expecter{mock: &_m.Mock}
}

// Execute provides a mock function with given fields: caps
func (_m *GetHashFunc) Execute(caps capabilities.Capabilities) []byte {
	ret := _m.Called(caps)

	if len(ret) == 0 {
		panic("no return value specified for Execute")
	}

	var r0 []byte
	if rf, ok := ret.Get(0).(func(capabilities.Capabilities) []byte); ok {
		r0 = rf(caps)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0
}

// GetHashFunc_Execute_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Execute'
type GetHashFunc_Execute_Call struct {
	*mock.Call
}

// Execute is a helper method to define mock.On call
//   - caps capabilities.Capabilities
func (_e *GetHashFunc_Expecter) Execute(caps interface{}) *GetHashFunc_Execute_Call {
	return &GetHashFunc_Execute_Call{Call: _e.mock.On("Execute", caps)}
}

func (_c *GetHashFunc_Execute_Call) Run(run func(caps capabilities.Capabilities)) *GetHashFunc_Execute_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(capabilities.Capabilities))
	})
	return _c
}

func (_c *GetHashFunc_Execute_Call) Return(_a0 []byte) *GetHashFunc_Execute_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *GetHashFunc_Execute_Call) RunAndReturn(run func(capabilities.Capabilities) []byte) *GetHashFunc_Execute_Call {
	_c.Call.Return(run)
	return _c
}

// NewGetHashFunc creates a new instance of GetHashFunc. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewGetHashFunc(t interface {
	mock.TestingT
	Cleanup(func())
}) *GetHashFunc {
	mock := &GetHashFunc{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
