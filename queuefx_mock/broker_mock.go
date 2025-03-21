// Code generated by mockery v2.52.3. DO NOT EDIT.

package queuefx_mock

import (
	context "context"

	queuefx "github.com/eser/ajan/queuefx"
	mock "github.com/stretchr/testify/mock"
)

// Broker is an autogenerated mock type for the Broker type
type Broker struct {
	mock.Mock
}

type Broker_Expecter struct {
	mock *mock.Mock
}

func (_m *Broker) EXPECT() *Broker_Expecter {
	return &Broker_Expecter{mock: &_m.Mock}
}

// GetDialect provides a mock function with no fields
func (_m *Broker) GetDialect() queuefx.Dialect {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetDialect")
	}

	var r0 queuefx.Dialect
	if rf, ok := ret.Get(0).(func() queuefx.Dialect); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(queuefx.Dialect)
	}

	return r0
}

// Broker_GetDialect_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetDialect'
type Broker_GetDialect_Call struct {
	*mock.Call
}

// GetDialect is a helper method to define mock.On call
func (_e *Broker_Expecter) GetDialect() *Broker_GetDialect_Call {
	return &Broker_GetDialect_Call{Call: _e.mock.On("GetDialect")}
}

func (_c *Broker_GetDialect_Call) Run(run func()) *Broker_GetDialect_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Broker_GetDialect_Call) Return(_a0 queuefx.Dialect) *Broker_GetDialect_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Broker_GetDialect_Call) RunAndReturn(run func() queuefx.Dialect) *Broker_GetDialect_Call {
	_c.Call.Return(run)
	return _c
}

// Publish provides a mock function with given fields: ctx, name, body
func (_m *Broker) Publish(ctx context.Context, name string, body []byte) error {
	ret := _m.Called(ctx, name, body)

	if len(ret) == 0 {
		panic("no return value specified for Publish")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []byte) error); ok {
		r0 = rf(ctx, name, body)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Broker_Publish_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Publish'
type Broker_Publish_Call struct {
	*mock.Call
}

// Publish is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - body []byte
func (_e *Broker_Expecter) Publish(ctx interface{}, name interface{}, body interface{}) *Broker_Publish_Call {
	return &Broker_Publish_Call{Call: _e.mock.On("Publish", ctx, name, body)}
}

func (_c *Broker_Publish_Call) Run(run func(ctx context.Context, name string, body []byte)) *Broker_Publish_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].([]byte))
	})
	return _c
}

func (_c *Broker_Publish_Call) Return(_a0 error) *Broker_Publish_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Broker_Publish_Call) RunAndReturn(run func(context.Context, string, []byte) error) *Broker_Publish_Call {
	_c.Call.Return(run)
	return _c
}

// QueueDeclare provides a mock function with given fields: ctx, name
func (_m *Broker) QueueDeclare(ctx context.Context, name string) (string, error) {
	ret := _m.Called(ctx, name)

	if len(ret) == 0 {
		panic("no return value specified for QueueDeclare")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (string, error)); ok {
		return rf(ctx, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) string); ok {
		r0 = rf(ctx, name)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Broker_QueueDeclare_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'QueueDeclare'
type Broker_QueueDeclare_Call struct {
	*mock.Call
}

// QueueDeclare is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
func (_e *Broker_Expecter) QueueDeclare(ctx interface{}, name interface{}) *Broker_QueueDeclare_Call {
	return &Broker_QueueDeclare_Call{Call: _e.mock.On("QueueDeclare", ctx, name)}
}

func (_c *Broker_QueueDeclare_Call) Run(run func(ctx context.Context, name string)) *Broker_QueueDeclare_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Broker_QueueDeclare_Call) Return(_a0 string, _a1 error) *Broker_QueueDeclare_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Broker_QueueDeclare_Call) RunAndReturn(run func(context.Context, string) (string, error)) *Broker_QueueDeclare_Call {
	_c.Call.Return(run)
	return _c
}

// NewBroker creates a new instance of Broker. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBroker(t interface {
	mock.TestingT
	Cleanup(func())
}) *Broker {
	mock := &Broker{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
