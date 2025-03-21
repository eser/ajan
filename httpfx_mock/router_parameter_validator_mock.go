// Code generated by mockery v2.52.3. DO NOT EDIT.

package httpfx_mock

import mock "github.com/stretchr/testify/mock"

// RouterParameterValidator is an autogenerated mock type for the RouterParameterValidator type
type RouterParameterValidator struct {
	mock.Mock
}

type RouterParameterValidator_Expecter struct {
	mock *mock.Mock
}

func (_m *RouterParameterValidator) EXPECT() *RouterParameterValidator_Expecter {
	return &RouterParameterValidator_Expecter{mock: &_m.Mock}
}

// Execute provides a mock function with given fields: inputString
func (_m *RouterParameterValidator) Execute(inputString string) (string, error) {
	ret := _m.Called(inputString)

	if len(ret) == 0 {
		panic("no return value specified for Execute")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (string, error)); ok {
		return rf(inputString)
	}
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(inputString)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(inputString)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RouterParameterValidator_Execute_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Execute'
type RouterParameterValidator_Execute_Call struct {
	*mock.Call
}

// Execute is a helper method to define mock.On call
//   - inputString string
func (_e *RouterParameterValidator_Expecter) Execute(inputString interface{}) *RouterParameterValidator_Execute_Call {
	return &RouterParameterValidator_Execute_Call{Call: _e.mock.On("Execute", inputString)}
}

func (_c *RouterParameterValidator_Execute_Call) Run(run func(inputString string)) *RouterParameterValidator_Execute_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *RouterParameterValidator_Execute_Call) Return(_a0 string, _a1 error) *RouterParameterValidator_Execute_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *RouterParameterValidator_Execute_Call) RunAndReturn(run func(string) (string, error)) *RouterParameterValidator_Execute_Call {
	_c.Call.Return(run)
	return _c
}

// NewRouterParameterValidator creates a new instance of RouterParameterValidator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRouterParameterValidator(t interface {
	mock.TestingT
	Cleanup(func())
}) *RouterParameterValidator {
	mock := &RouterParameterValidator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
