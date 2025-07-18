// Code generated by mockery v2.52.3. DO NOT EDIT.

package mocks

import (
	echo "github.com/labstack/echo/v4"
	mock "github.com/stretchr/testify/mock"
)

// WDSessionController is an autogenerated mock type for the WDSessionController type
type WDSessionController struct {
	mock.Mock
}

type WDSessionController_Expecter struct {
	mock *mock.Mock
}

func (_m *WDSessionController) EXPECT() *WDSessionController_Expecter {
	return &WDSessionController_Expecter{mock: &_m.Mock}
}

// CreateSession provides a mock function with given fields: ctx
func (_m *WDSessionController) CreateSession(ctx echo.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for CreateSession")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(echo.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WDSessionController_CreateSession_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateSession'
type WDSessionController_CreateSession_Call struct {
	*mock.Call
}

// CreateSession is a helper method to define mock.On call
//   - ctx echo.Context
func (_e *WDSessionController_Expecter) CreateSession(ctx interface{}) *WDSessionController_CreateSession_Call {
	return &WDSessionController_CreateSession_Call{Call: _e.mock.On("CreateSession", ctx)}
}

func (_c *WDSessionController_CreateSession_Call) Run(run func(ctx echo.Context)) *WDSessionController_CreateSession_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(echo.Context))
	})
	return _c
}

func (_c *WDSessionController_CreateSession_Call) Return(_a0 error) *WDSessionController_CreateSession_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *WDSessionController_CreateSession_Call) RunAndReturn(run func(echo.Context) error) *WDSessionController_CreateSession_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteSession provides a mock function with given fields: ctx
func (_m *WDSessionController) DeleteSession(ctx echo.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for DeleteSession")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(echo.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WDSessionController_DeleteSession_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteSession'
type WDSessionController_DeleteSession_Call struct {
	*mock.Call
}

// DeleteSession is a helper method to define mock.On call
//   - ctx echo.Context
func (_e *WDSessionController_Expecter) DeleteSession(ctx interface{}) *WDSessionController_DeleteSession_Call {
	return &WDSessionController_DeleteSession_Call{Call: _e.mock.On("DeleteSession", ctx)}
}

func (_c *WDSessionController_DeleteSession_Call) Run(run func(ctx echo.Context)) *WDSessionController_DeleteSession_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(echo.Context))
	})
	return _c
}

func (_c *WDSessionController_DeleteSession_Call) Return(_a0 error) *WDSessionController_DeleteSession_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *WDSessionController_DeleteSession_Call) RunAndReturn(run func(echo.Context) error) *WDSessionController_DeleteSession_Call {
	_c.Call.Return(run)
	return _c
}

// Status provides a mock function with given fields: ctx
func (_m *WDSessionController) Status(ctx echo.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Status")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(echo.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WDSessionController_Status_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Status'
type WDSessionController_Status_Call struct {
	*mock.Call
}

// Status is a helper method to define mock.On call
//   - ctx echo.Context
func (_e *WDSessionController_Expecter) Status(ctx interface{}) *WDSessionController_Status_Call {
	return &WDSessionController_Status_Call{Call: _e.mock.On("Status", ctx)}
}

func (_c *WDSessionController_Status_Call) Run(run func(ctx echo.Context)) *WDSessionController_Status_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(echo.Context))
	})
	return _c
}

func (_c *WDSessionController_Status_Call) Return(_a0 error) *WDSessionController_Status_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *WDSessionController_Status_Call) RunAndReturn(run func(echo.Context) error) *WDSessionController_Status_Call {
	_c.Call.Return(run)
	return _c
}

// ValidateSession provides a mock function with given fields: next
func (_m *WDSessionController) ValidateSession(next echo.HandlerFunc) echo.HandlerFunc {
	ret := _m.Called(next)

	if len(ret) == 0 {
		panic("no return value specified for ValidateSession")
	}

	var r0 echo.HandlerFunc
	if rf, ok := ret.Get(0).(func(echo.HandlerFunc) echo.HandlerFunc); ok {
		r0 = rf(next)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(echo.HandlerFunc)
		}
	}

	return r0
}

// WDSessionController_ValidateSession_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ValidateSession'
type WDSessionController_ValidateSession_Call struct {
	*mock.Call
}

// ValidateSession is a helper method to define mock.On call
//   - next echo.HandlerFunc
func (_e *WDSessionController_Expecter) ValidateSession(next interface{}) *WDSessionController_ValidateSession_Call {
	return &WDSessionController_ValidateSession_Call{Call: _e.mock.On("ValidateSession", next)}
}

func (_c *WDSessionController_ValidateSession_Call) Run(run func(next echo.HandlerFunc)) *WDSessionController_ValidateSession_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(echo.HandlerFunc))
	})
	return _c
}

func (_c *WDSessionController_ValidateSession_Call) Return(_a0 echo.HandlerFunc) *WDSessionController_ValidateSession_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *WDSessionController_ValidateSession_Call) RunAndReturn(run func(echo.HandlerFunc) echo.HandlerFunc) *WDSessionController_ValidateSession_Call {
	_c.Call.Return(run)
	return _c
}

// NewWDSessionController creates a new instance of WDSessionController. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewWDSessionController(t interface {
	mock.TestingT
	Cleanup(func())
}) *WDSessionController {
	mock := &WDSessionController{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
