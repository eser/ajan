package tracesfx

import (
	"errors"
	"fmt"
	"reflect"

	"go.opentelemetry.io/otel/sdk/trace"
)

var (
	ErrBridgeNoRegistryProvided      = errors.New("no registry provided")
	ErrBridgeConnectionNotFound      = errors.New("connection not found")
	ErrBridgeTraceExporterNotFound   = errors.New("trace exporter not found")
	ErrBridgeConnectionIsNil         = errors.New("connection is nil")
	ErrBridgeMissingGetTraceExporter = errors.New(
		"connection does not have GetTraceExporter method",
	)
	ErrBridgeNoTraceExporterAvailable = errors.New("no trace exporter available")
	ErrBridgeInvalidTraceExporter     = errors.New("returned value is not a trace exporter")
)

// ConnectionRegistry interface to avoid import cycle with connfx.
type ConnectionRegistry interface {
	GetNamed(name string) any
}

// OTLPBridge handles integration with OTLP connections from connfx.
type OTLPBridge struct {
	registry ConnectionRegistry
}

// NewOTLPBridge creates a new OTLP bridge.
func NewOTLPBridge(registry ConnectionRegistry) *OTLPBridge {
	return &OTLPBridge{
		registry: registry,
	}
}

// GetTraceExporter gets a trace exporter from an OTLP connection.
func (b *OTLPBridge) GetTraceExporter(connectionName string) (trace.SpanExporter, error) {
	if b.registry == nil {
		return nil, ErrBridgeNoRegistryProvided
	}

	conn := b.registry.GetNamed(connectionName)
	if conn == nil {
		return nil, fmt.Errorf("%w (name=%q)", ErrBridgeConnectionNotFound, connectionName)
	}

	// Use reflection to call methods on the OTLP connection
	return b.getTraceExporterViaReflection(conn)
}

// getTraceExporterViaReflection uses reflection to get trace exporter.
func (b *OTLPBridge) getTraceExporterViaReflection(conn any) (trace.SpanExporter, error) {
	// Check if connection has GetTraceExporter method
	connValue := reflect.ValueOf(conn)
	if connValue.IsNil() {
		return nil, ErrBridgeConnectionIsNil
	}

	getTraceExporterMethod := connValue.MethodByName("GetTraceExporter")
	if !getTraceExporterMethod.IsValid() {
		return nil, ErrBridgeMissingGetTraceExporter
	}

	// Call GetTraceExporter
	result := getTraceExporterMethod.Call(nil)
	if len(result) == 0 || result[0].IsNil() {
		return nil, ErrBridgeNoTraceExporterAvailable
	}

	exporter, ok := result[0].Interface().(trace.SpanExporter)
	if !ok {
		return nil, ErrBridgeInvalidTraceExporter
	}

	return exporter, nil
}
