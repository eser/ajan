package metricsfx

import (
	"errors"
	"fmt"
	"reflect"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	ErrNoRegistryProvided        = errors.New("no registry provided")
	ErrConnectionNotFound        = errors.New("connection not found")
	ErrMetricExporterNotFound    = errors.New("metric exporter not found")
	ErrRegistryMissingGetNamed   = errors.New("registry does not have GetNamed method")
	ErrConnectionIsNil           = errors.New("connection is nil")
	ErrMissingGetMetricExporter  = errors.New("connection does not have GetMetricExporter method")
	ErrNoMetricExporterAvailable = errors.New("no metric exporter available")
	ErrInvalidMetricExporter     = errors.New("returned value is not a metric exporter")
)

// ConnectionRegistry interface to avoid import cycle with connfx.
type ConnectionRegistry interface {
	GetNamed(name string) any
}

// OTLPBridge handles integration with OTLP connections from connfx.
type OTLPBridge struct {
	registry any // ConnectionRegistry interface{}
}

// NewOTLPBridge creates a new OTLP bridge.
func NewOTLPBridge(registry any) *OTLPBridge {
	return &OTLPBridge{
		registry: registry,
	}
}

// GetMetricExporter gets metric exporter from OTLP connection using reflection.
func (b *OTLPBridge) GetMetricExporter(connectionName string) (sdkmetric.Exporter, error) {
	if b.registry == nil {
		return nil, ErrNoRegistryProvided
	}

	conn, err := b.getConnectionViaReflection(connectionName)
	if err != nil {
		return nil, fmt.Errorf("%w (name=%q): %w", ErrConnectionNotFound, connectionName, err)
	}

	exporter, err := b.getMetricExporterViaReflection(conn)
	if err != nil {
		return nil, fmt.Errorf("%w (name=%q): %w", ErrMetricExporterNotFound, connectionName, err)
	}

	return exporter, nil
}

// getConnectionViaReflection uses reflection to get connection from registry.
func (b *OTLPBridge) getConnectionViaReflection(connectionName string) (any, error) {
	registryValue := reflect.ValueOf(b.registry)
	if registryValue.IsNil() {
		return nil, ErrNoRegistryProvided
	}

	getNamedMethod := registryValue.MethodByName("GetNamed")
	if !getNamedMethod.IsValid() {
		return nil, fmt.Errorf("%w", ErrRegistryMissingGetNamed)
	}

	// Call GetNamed with connection name
	result := getNamedMethod.Call([]reflect.Value{reflect.ValueOf(connectionName)})
	if len(result) == 0 || result[0].IsNil() {
		return nil, fmt.Errorf("%w", ErrConnectionNotFound)
	}

	return result[0].Interface(), nil
}

// getMetricExporterViaReflection uses reflection to get metric exporter.
func (b *OTLPBridge) getMetricExporterViaReflection(conn any) (sdkmetric.Exporter, error) {
	// Check if connection has GetMetricExporter method
	connValue := reflect.ValueOf(conn)
	if connValue.IsNil() {
		return nil, fmt.Errorf("%w", ErrConnectionIsNil)
	}

	getMetricExporterMethod := connValue.MethodByName("GetMetricExporter")
	if !getMetricExporterMethod.IsValid() {
		return nil, fmt.Errorf("%w", ErrMissingGetMetricExporter)
	}

	// Call GetMetricExporter
	result := getMetricExporterMethod.Call(nil)
	if len(result) == 0 || result[0].IsNil() {
		return nil, fmt.Errorf("%w", ErrNoMetricExporterAvailable)
	}

	exporter, ok := result[0].Interface().(sdkmetric.Exporter)
	if !ok {
		return nil, fmt.Errorf("%w", ErrInvalidMetricExporter)
	}

	return exporter, nil
}
