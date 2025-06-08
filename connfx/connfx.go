package connfx

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/eser/ajan/logfx"
)

// Sentinel errors for connfx package.
var (
	ErrProtocolMismatch     = errors.New("protocol mismatch")
	ErrBehaviorNotSupported = errors.New("behavior not supported")
)

// Manager provides a convenient wrapper around the connection registry.
type Manager struct {
	registry *Registry
}

// NewManager creates a new connection manager
// Note: Factories must be registered separately using RegisterAdapter.
func NewManager(logger *logfx.Logger) *Manager {
	registry := NewRegistry(logger)

	return &Manager{
		registry: registry,
	}
}

// RegisterAdapter registers a connection adapter (factory) for a protocol.
func (m *Manager) RegisterAdapter(factory ConnectionFactory) error {
	return m.registry.RegisterFactory(factory)
}

// GetRegistry returns the underlying registry.
func (m *Manager) GetRegistry() *Registry {
	return m.registry
}

// LoadFromConfig loads connections from configuration.
func (m *Manager) LoadFromConfig(ctx context.Context, config *Config) error {
	return m.registry.LoadFromConfig(ctx, config)
}

// Connection retrieval by behavior

// GetStatefulConnections returns all stateful connections.
func (m *Manager) GetStatefulConnections() []Connection {
	return m.registry.GetByBehavior(ConnectionBehaviorStateful)
}

// GetStatelessConnections returns all stateless connections.
func (m *Manager) GetStatelessConnections() []Connection {
	return m.registry.GetByBehavior(ConnectionBehaviorStateless)
}

// GetStreamingConnections returns all streaming connections.
func (m *Manager) GetStreamingConnections() []Connection {
	return m.registry.GetByBehavior(ConnectionBehaviorStreaming)
}

// Connection retrieval by protocol

// GetConnectionsByProtocol returns all connections for a specific protocol.
func (m *Manager) GetConnectionsByProtocol(protocol string) []Connection {
	return m.registry.GetByProtocol(protocol)
}

// Connection retrieval by name with type safety

// GetConnection returns a connection by name.
func (m *Manager) GetConnection(name string) (Connection, error) {
	conn := m.registry.GetNamed(name)
	if conn == nil {
		return nil, fmt.Errorf("%w (name=%q)", ErrConnectionNotFound, name)
	}

	return conn, nil
}

// GetDefaultConnection returns the default connection.
func (m *Manager) GetDefaultConnection() (Connection, error) {
	return m.GetConnection(DefaultConnection)
}

// GetConnectionByProtocol returns a named connection and verifies it uses the expected protocol.
func (m *Manager) GetConnectionByProtocol(name, expectedProtocol string) (Connection, error) {
	conn, err := m.GetConnection(name)
	if err != nil {
		return nil, err
	}

	if conn.GetProtocol() != expectedProtocol {
		return nil, fmt.Errorf("%w (name=%q, expected=%q, got=%q)",
			ErrProtocolMismatch, name, expectedProtocol, conn.GetProtocol())
	}

	return conn, nil
}

// GetConnectionByBehavior returns a named connection and verifies it has the expected behavior.
func (m *Manager) GetConnectionByBehavior(
	name string,
	expectedBehavior ConnectionBehavior,
) (Connection, error) {
	conn, err := m.GetConnection(name)
	if err != nil {
		return nil, err
	}

	// Check if connection supports the expected behavior
	if slices.Contains(conn.GetBehaviors(), expectedBehavior) {
		return conn, nil
	}

	return nil, fmt.Errorf("%w (name=%q, expected=%q, supported=%v)",
		ErrBehaviorNotSupported, name, expectedBehavior, conn.GetBehaviors())
}

// Generic connection helpers

// AddConnection adds a new connection.
func (m *Manager) AddConnection(ctx context.Context, config ConnectionConfig) error {
	return m.registry.AddConnection(ctx, config)
}

// RemoveConnection removes a connection.
func (m *Manager) RemoveConnection(ctx context.Context, name string) error {
	return m.registry.RemoveConnection(ctx, name)
}

// ListConnections returns all connection names.
func (m *Manager) ListConnections() []string {
	return m.registry.ListConnections()
}

// ListRegisteredProtocols returns all registered protocols.
func (m *Manager) ListRegisteredProtocols() []string {
	return m.registry.ListRegisteredProtocols()
}

// HealthCheck performs health checks on all connections.
func (m *Manager) HealthCheck(ctx context.Context) map[string]*HealthStatus {
	return m.registry.HealthCheck(ctx)
}

// HealthCheckNamed performs a health check on a specific connection.
func (m *Manager) HealthCheckNamed(ctx context.Context, name string) (*HealthStatus, error) {
	return m.registry.HealthCheckNamed(ctx, name)
}

// Close closes all connections.
func (m *Manager) Close(ctx context.Context) error {
	return m.registry.Close(ctx)
}

// Package-level convenience functions for global usage

var defaultManager *Manager //nolint:gochecknoglobals

// Initialize sets up the default connection manager.
func Initialize(logger *logfx.Logger) {
	defaultManager = NewManager(logger)
}

// RegisterAdapter registers an adapter with the default manager.
func RegisterAdapter(factory ConnectionFactory) error {
	return Default().RegisterAdapter(factory)
}

// Default returns the default connection manager.
func Default() *Manager {
	if defaultManager == nil {
		panic("connfx: default manager not initialized, call connfx.Initialize() first")
	}

	return defaultManager
}

// LoadConfig loads connections from configuration using the default manager.
func LoadConfig(ctx context.Context, config *Config) error {
	return Default().LoadFromConfig(ctx, config)
}

// GetConnection returns a connection by name from the default manager.
func GetConnection(name string) (Connection, error) {
	return Default().GetConnection(name)
}

// GetStatefulConnections returns all stateful connections from the default manager.
func GetStatefulConnections() []Connection {
	return Default().GetStatefulConnections()
}

// GetStatelessConnections returns all stateless connections from the default manager.
func GetStatelessConnections() []Connection {
	return Default().GetStatelessConnections()
}

// GetStreamingConnections returns all streaming connections from the default manager.
func GetStreamingConnections() []Connection {
	return Default().GetStreamingConnections()
}

// HealthCheck performs health checks on all connections using the default manager.
func HealthCheck(ctx context.Context) map[string]*HealthStatus {
	return Default().HealthCheck(ctx)
}
