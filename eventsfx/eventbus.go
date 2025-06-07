package eventsfx

import (
	"errors"
	"fmt"
	"time"

	"github.com/eser/ajan/logfx"
	"go.opentelemetry.io/otel/metric"
)

type Event struct {
	Data      any
	ReplyChan chan any
	Name      string
}

type EventHandler func(event Event)

type EventBus struct {
	InnerMetrics *Metrics

	Subscribers map[string][]EventHandler
	Queue       chan Event

	Config *Config
	logger *logfx.Logger
}

var ErrEventTimeout = errors.New("event timeout")

type MetricsProvider interface {
	GetMeterProvider() metric.MeterProvider
}

func NewEventBus(
	config *Config,
	metricsProvider MetricsProvider,
	logger *logfx.Logger,
) *EventBus {
	return &EventBus{
		InnerMetrics: NewMetrics(metricsProvider),

		Subscribers: make(map[string][]EventHandler),
		Queue:       make(chan Event, config.DefaultBufferSize),

		Config: config,
		logger: logger,
	}
}

func (bus *EventBus) Subscribe(eventName string, handler EventHandler) {
	bus.Subscribers[eventName] = append(bus.Subscribers[eventName], handler)
}

func (bus *EventBus) Publish(event Event) {
	bus.Queue <- event
}

func (bus *EventBus) PublishSync(event Event) (any, error) {
	replyChan := make(chan any)

	event.ReplyChan = replyChan
	bus.Publish(event)

	select {
	case result := <-replyChan:
		return result, nil
	case <-time.After(bus.Config.ReplyTimeout):
		return nil, fmt.Errorf("%w - %q", ErrEventTimeout, event.Name)
	}
}

func (bus *EventBus) Dispatch(event Event) {
	if handlers, ok := bus.Subscribers[event.Name]; ok {
		for _, handler := range handlers {
			go handler(event)
		}
	}
}

func (bus *EventBus) DispatchSync(event Event) {
	if handlers, ok := bus.Subscribers[event.Name]; ok {
		for _, handler := range handlers {
			handler(event)
		}
	}
}

func (bus *EventBus) Listen() {
	for event := range bus.Queue {
		bus.Dispatch(event)
	}
}

func (bus *EventBus) Cleanup() {
	close(bus.Queue)
}
