// Code generated by mockery v2.52.3. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// BrowserConverterConfig is an autogenerated mock type for the BrowserConverterConfig type
type BrowserConverterConfig struct {
	mock.Mock
}

type BrowserConverterConfig_Expecter struct {
	mock *mock.Mock
}

func (_m *BrowserConverterConfig) EXPECT() *BrowserConverterConfig_Expecter {
	return &BrowserConverterConfig_Expecter{mock: &_m.Mock}
}

// JobID provides a mock function with no fields
func (_m *BrowserConverterConfig) JobID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for JobID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// BrowserConverterConfig_JobID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'JobID'
type BrowserConverterConfig_JobID_Call struct {
	*mock.Call
}

// JobID is a helper method to define mock.On call
func (_e *BrowserConverterConfig_Expecter) JobID() *BrowserConverterConfig_JobID_Call {
	return &BrowserConverterConfig_JobID_Call{Call: _e.mock.On("JobID")}
}

func (_c *BrowserConverterConfig_JobID_Call) Run(run func()) *BrowserConverterConfig_JobID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *BrowserConverterConfig_JobID_Call) Return(_a0 string) *BrowserConverterConfig_JobID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BrowserConverterConfig_JobID_Call) RunAndReturn(run func() string) *BrowserConverterConfig_JobID_Call {
	_c.Call.Return(run)
	return _c
}

// Lineage provides a mock function with no fields
func (_m *BrowserConverterConfig) Lineage() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Lineage")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// BrowserConverterConfig_Lineage_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Lineage'
type BrowserConverterConfig_Lineage_Call struct {
	*mock.Call
}

// Lineage is a helper method to define mock.On call
func (_e *BrowserConverterConfig_Expecter) Lineage() *BrowserConverterConfig_Lineage_Call {
	return &BrowserConverterConfig_Lineage_Call{Call: _e.mock.On("Lineage")}
}

func (_c *BrowserConverterConfig_Lineage_Call) Run(run func()) *BrowserConverterConfig_Lineage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *BrowserConverterConfig_Lineage_Call) Return(_a0 string) *BrowserConverterConfig_Lineage_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BrowserConverterConfig_Lineage_Call) RunAndReturn(run func() string) *BrowserConverterConfig_Lineage_Call {
	_c.Call.Return(run)
	return _c
}

// ProjectName provides a mock function with no fields
func (_m *BrowserConverterConfig) ProjectName() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ProjectName")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// BrowserConverterConfig_ProjectName_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ProjectName'
type BrowserConverterConfig_ProjectName_Call struct {
	*mock.Call
}

// ProjectName is a helper method to define mock.On call
func (_e *BrowserConverterConfig_Expecter) ProjectName() *BrowserConverterConfig_ProjectName_Call {
	return &BrowserConverterConfig_ProjectName_Call{Call: _e.mock.On("ProjectName")}
}

func (_c *BrowserConverterConfig_ProjectName_Call) Run(run func()) *BrowserConverterConfig_ProjectName_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *BrowserConverterConfig_ProjectName_Call) Return(_a0 string) *BrowserConverterConfig_ProjectName_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BrowserConverterConfig_ProjectName_Call) RunAndReturn(run func() string) *BrowserConverterConfig_ProjectName_Call {
	_c.Call.Return(run)
	return _c
}

// ProjectNamespace provides a mock function with no fields
func (_m *BrowserConverterConfig) ProjectNamespace() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ProjectNamespace")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// BrowserConverterConfig_ProjectNamespace_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ProjectNamespace'
type BrowserConverterConfig_ProjectNamespace_Call struct {
	*mock.Call
}

// ProjectNamespace is a helper method to define mock.On call
func (_e *BrowserConverterConfig_Expecter) ProjectNamespace() *BrowserConverterConfig_ProjectNamespace_Call {
	return &BrowserConverterConfig_ProjectNamespace_Call{Call: _e.mock.On("ProjectNamespace")}
}

func (_c *BrowserConverterConfig_ProjectNamespace_Call) Run(run func()) *BrowserConverterConfig_ProjectNamespace_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *BrowserConverterConfig_ProjectNamespace_Call) Return(_a0 string) *BrowserConverterConfig_ProjectNamespace_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BrowserConverterConfig_ProjectNamespace_Call) RunAndReturn(run func() string) *BrowserConverterConfig_ProjectNamespace_Call {
	_c.Call.Return(run)
	return _c
}

// NewBrowserConverterConfig creates a new instance of BrowserConverterConfig. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBrowserConverterConfig(t interface {
	mock.TestingT
	Cleanup(func())
}) *BrowserConverterConfig {
	mock := &BrowserConverterConfig{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
