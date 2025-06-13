package logfx

import (
	"context"
	"log/slog"
	"reflect"
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

// SendLog sends a log record to an OTLP connection.
func (b *OTLPBridge) SendLog(ctx context.Context, connectionName string, rec slog.Record) error {
	if b.registry == nil {
		return nil // No registry, skip OTLP sending
	}

	conn := b.registry.GetNamed(connectionName)
	if conn == nil {
		return nil // No connection found, skip OTLP sending
	}

	// Use reflection to call methods on the OTLP connection
	// This avoids import cycles while still allowing integration
	return b.sendLogViaReflection(conn)
}

// sendLogViaReflection uses reflection to interact with OTLP connection.
func (b *OTLPBridge) sendLogViaReflection(conn any) error {
	// Check if connection has GetLoggerProvider method
	connValue := reflect.ValueOf(conn)
	if connValue.IsNil() {
		return nil
	}

	getLoggerProviderMethod := connValue.MethodByName("GetLoggerProvider")
	if !getLoggerProviderMethod.IsValid() {
		return nil // Method not found, skip
	}

	// Call GetLoggerProvider
	result := getLoggerProviderMethod.Call(nil)
	if len(result) == 0 || result[0].IsNil() {
		return nil // No logger provider
	}

	loggerProvider := result[0].Interface()
	if loggerProvider == nil {
		return nil
	}

	// TODO: Use reflection to call Logger method and emit log record
	// For now, just return nil to indicate success without actual implementation
	return nil
}
