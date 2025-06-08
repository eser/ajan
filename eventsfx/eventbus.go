package eventsfx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eser/ajan/logfx"
	"github.com/eser/ajan/metricsfx"
)

var (
	ErrEventTimeout          = errors.New("event timeout")
	ErrFailedToCreateMetrics = errors.New("failed to create event metrics")
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

func NewEventBus(
	config *Config,
	metricsProvider *metricsfx.MetricsProvider,
	logger *logfx.Logger,
) (*EventBus, error) {
	metrics, err := NewMetrics(metricsProvider)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToCreateMetrics, err)
	}

	return &EventBus{
		InnerMetrics: metrics,

		Subscribers: make(map[string][]EventHandler),
		Queue:       make(chan Event, config.DefaultBufferSize),

		Config: config,
		logger: logger,
	}, nil
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
		return nil, fmt.Errorf("%w (event_name=%q)", ErrEventTimeout, event.Name)
	}
}

func (bus *EventBus) Dispatch(event Event) {
	if handlers, ok := bus.Subscribers[event.Name]; ok {
		for _, handler := range handlers {
			go handler(event)
		}

		// Record metrics for each dispatch
		ctx := context.Background()
		attrs := []metricsfx.Attribute{
			metricsfx.StringAttr("event_name", event.Name),
			metricsfx.IntAttr("handler_count", len(handlers)),
		}
		bus.InnerMetrics.EventDispatchesTotal.Inc(ctx, attrs...)
	}
}

func (bus *EventBus) DispatchSync(event Event) {
	if handlers, ok := bus.Subscribers[event.Name]; ok {
		for _, handler := range handlers {
			handler(event)
		}

		// Record metrics for each dispatch
		ctx := context.Background()
		attrs := []metricsfx.Attribute{
			metricsfx.StringAttr("event_name", event.Name),
			metricsfx.IntAttr("handler_count", len(handlers)),
		}
		bus.InnerMetrics.EventDispatchesTotal.Inc(ctx, attrs...)
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
