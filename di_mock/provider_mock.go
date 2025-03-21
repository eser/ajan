// Code generated by mockery v2.52.3. DO NOT EDIT.

package di_mock

import mock "github.com/stretchr/testify/mock"

// Provider is an autogenerated mock type for the Provider type
type Provider struct {
	mock.Mock
}

type Provider_Expecter struct {
	mock *mock.Mock
}

func (_m *Provider) EXPECT() *Provider_Expecter {
	return &Provider_Expecter{mock: &_m.Mock}
}

// Execute provides a mock function with given fields: args
func (_m *Provider) Execute(args []any) any {
	ret := _m.Called(args)

	if len(ret) == 0 {
		panic("no return value specified for Execute")
	}

	var r0 any
	if rf, ok := ret.Get(0).(func([]any) any); ok {
		r0 = rf(args)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(any)
		}
	}

	return r0
}

// Provider_Execute_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Execute'
type Provider_Execute_Call struct {
	*mock.Call
}

// Execute is a helper method to define mock.On call
//   - args []any
func (_e *Provider_Expecter) Execute(args interface{}) *Provider_Execute_Call {
	return &Provider_Execute_Call{Call: _e.mock.On("Execute", args)}
}

func (_c *Provider_Execute_Call) Run(run func(args []any)) *Provider_Execute_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]any))
	})
	return _c
}

func (_c *Provider_Execute_Call) Return(_a0 any) *Provider_Execute_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Provider_Execute_Call) RunAndReturn(run func([]any) any) *Provider_Execute_Call {
	_c.Call.Return(run)
	return _c
}

// NewProvider creates a new instance of Provider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewProvider(t interface {
	mock.TestingT
	Cleanup(func())
}) *Provider {
	mock := &Provider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
