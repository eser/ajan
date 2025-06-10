package connfx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/eser/ajan/logfx"
)

var (
	ErrConnectionNotFound       = errors.New("connection not found")
	ErrConnectionAlreadyExists  = errors.New("connection already exists")
	ErrFailedToCreateConnection = errors.New("failed to create connection")
	ErrUnsupportedProtocol      = errors.New("unsupported protocol")
	ErrFailedToCloseConnections = errors.New("failed to close connections")
	ErrFailedToAddConnection    = errors.New("failed to add connection")
	ErrConnectionNotSupported   = errors.New("connection does not support required operations")
	ErrInterfaceNotImplemented  = errors.New("connection does not implement required interface")
)

const DefaultConnection = "default"

// Registry manages all connections in the system.
type Registry struct {
	connections map[string]Connection
	factories   map[string]ConnectionFactory // protocol -> factory
	logger      *logfx.Logger
	mu          sync.RWMutex
}

// NewRegistry creates a new connection registry.
func NewRegistry(logger *logfx.Logger) *Registry {
	return &Registry{
		connections: make(map[string]Connection),
		factories:   make(map[string]ConnectionFactory),
		logger:      logger,
		mu:          sync.RWMutex{},
	}
}

// RegisterFactory registers a connection factory for a specific protocol.
func (registry *Registry) RegisterFactory(factory ConnectionFactory) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	protocol := factory.GetProtocol()

	registry.factories[protocol] = factory

	behaviors := factory.GetSupportedBehaviors()
	behaviorStrs := make([]string, len(behaviors))

	for i, b := range behaviors {
		behaviorStrs[i] = string(b)
	}

	registry.logger.Info(
		"registered connection factory",
		slog.String("protocol", protocol),
		slog.Any("behaviors", behaviorStrs),
	)
}

// GetDefault returns the default connection.
func (registry *Registry) GetDefault() Connection {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return registry.connections[DefaultConnection]
}

// GetNamed returns a named connection.
func (registry *Registry) GetNamed(name string) Connection {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return registry.connections[name]
}

// GetByBehavior returns all connections of a specific behavior.
func (registry *Registry) GetByBehavior(behavior ConnectionBehavior) []Connection {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	var connections []Connection
	for _, conn := range registry.connections {
		// Check if the connection supports the requested behavior
		if slices.Contains(conn.GetBehaviors(), behavior) {
			connections = append(connections, conn)
		}
	}

	return connections
}

// GetByProtocol returns all connections of a specific protocol.
func (registry *Registry) GetByProtocol(protocol string) []Connection {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	var connections []Connection
	for _, conn := range registry.connections {
		if conn.GetProtocol() == protocol {
			connections = append(connections, conn)
		}
	}

	return connections
}

// ListConnections returns all connection names.
func (registry *Registry) ListConnections() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	names := make([]string, 0, len(registry.connections))
	for name := range registry.connections {
		names = append(names, name)
	}

	return names
}

// ListRegisteredProtocols returns all registered protocols.
func (registry *Registry) ListRegisteredProtocols() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	protocols := make([]string, 0, len(registry.factories))
	for protocol := range registry.factories {
		protocols = append(protocols, protocol)
	}

	return protocols
}

// AddConnection adds a new connection to the registry.
func (registry *Registry) AddConnection(
	ctx context.Context,
	name string,
	config *ConfigTarget,
) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	// Check if connection already exists
	if _, exists := registry.connections[name]; exists {
		return fmt.Errorf("%w (name=%q)", ErrConnectionAlreadyExists, name)
	}

	// Get factory for this protocol
	factory, exists := registry.factories[config.Protocol]
	if !exists {
		return fmt.Errorf("%w (protocol=%q)", ErrUnsupportedProtocol, config.Protocol)
	}

	registry.logger.Info(
		"creating connection",
		slog.String("name", name),
		slog.String("protocol", config.Protocol),
	)

	// Create the connection
	conn, err := factory.CreateConnection(ctx, config)
	if err != nil {
		registry.logger.Error(
			"failed to create connection",
			slog.String("error", err.Error()),
			slog.String("name", name),
			slog.String("protocol", config.Protocol),
		)

		return fmt.Errorf("%w (name=%q): %w", ErrFailedToCreateConnection, name, err)
	}

	registry.connections[name] = conn

	// Log the behaviors supported by this connection
	behaviors := conn.GetBehaviors()
	behaviorStrs := make([]string, len(behaviors))

	for i, b := range behaviors {
		behaviorStrs[i] = string(b)
	}

	registry.logger.Info(
		"successfully added connection",
		slog.String("name", name),
		slog.String("protocol", config.Protocol),
		slog.Any("behaviors", behaviorStrs),
	)

	return nil
}

// RemoveConnection removes a connection from the registry.
func (registry *Registry) RemoveConnection(ctx context.Context, name string) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	conn, exists := registry.connections[name]
	if !exists {
		return fmt.Errorf("%w (name=%q)", ErrConnectionNotFound, name)
	}

	// Close the connection
	if err := conn.Close(ctx); err != nil {
		registry.logger.Warn(
			"error closing connection",
			slog.String("error", err.Error()),
			slog.String("name", name),
		)
	}

	delete(registry.connections, name)

	registry.logger.Info(
		"removed connection",
		slog.String("name", name),
	)

	return nil
}

func (registry *Registry) LoadFromConfig(ctx context.Context, config *Config) error {
	for name, target := range config.Targets {
		if err := registry.AddConnection(ctx, name, &target); err != nil {
			return fmt.Errorf("%w (name=%q): %w", ErrFailedToAddConnection, name, err)
		}
	}

	return nil
}

// HealthCheck performs health checks on all connections.
func (registry *Registry) HealthCheck(ctx context.Context) map[string]*HealthStatus {
	registry.mu.RLock()

	connections := make(map[string]Connection, len(registry.connections))
	maps.Copy(connections, registry.connections)
	registry.mu.RUnlock()

	results := make(map[string]*HealthStatus)

	// Use a channel to collect results
	type healthResult struct {
		status *HealthStatus
		name   string
	}

	resultChan := make(chan healthResult, len(connections))

	// Perform health checks concurrently
	for name, conn := range connections {
		go func(name string, conn Connection) {
			status := conn.HealthCheck(ctx)
			resultChan <- healthResult{name: name, status: status}
		}(name, conn)
	}

	// Collect results
	for range len(connections) {
		result := <-resultChan
		results[result.name] = result.status
	}

	return results
}

// HealthCheckNamed performs a health check on a specific connection.
func (registry *Registry) HealthCheckNamed(
	ctx context.Context,
	name string,
) (*HealthStatus, error) {
	registry.mu.RLock()
	conn, exists := registry.connections[name]
	registry.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w (name=%q)", ErrConnectionNotFound, name)
	}

	return conn.HealthCheck(ctx), nil
}

// Close closes all connections in the registry.
func (registry *Registry) Close(ctx context.Context) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	var errors []error

	for name, conn := range registry.connections {
		if err := conn.Close(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to close connection %q: %w", name, err))
		}
	}

	// Clear the connections map
	registry.connections = make(map[string]Connection)

	if len(errors) > 0 {
		// Combine all errors into one
		errMsg := "errors closing connections: " + strings.Join(func() []string {
			errStrs := make([]string, len(errors))
			for i, err := range errors {
				errStrs[i] = err.Error()
			}

			return errStrs
		}(), "; ")

		return fmt.Errorf("%w: %s", ErrFailedToCloseConnections, errMsg)
	}

	return nil
}

// GetDataRepository returns a DataRepository from a connection if it supports it.
func (registry *Registry) GetDataRepository(name string) (DataRepository, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	conn := registry.connections[name]
	if conn == nil {
		return nil, fmt.Errorf("%w (name=%q)", ErrConnectionNotFound, name)
	}

	// Check if the connection supports key-value behavior
	behaviors := conn.GetBehaviors()
	if !slices.Contains(behaviors, ConnectionBehaviorKeyValue) &&
		!slices.Contains(behaviors, ConnectionBehaviorDocument) &&
		!slices.Contains(behaviors, ConnectionBehaviorRelational) {
		return nil, fmt.Errorf("%w (name=%q, operation=%q)",
			ErrConnectionNotSupported, name, "data repository operations")
	}

	// Try to get the repository from the raw connection
	repo, ok := conn.GetRawConnection().(DataRepository)
	if !ok {
		return nil, fmt.Errorf("%w (name=%q, interface=%q)",
			ErrInterfaceNotImplemented, name, "DataRepository")
	}

	return repo, nil
}

// GetTransactionalRepository returns a TransactionalRepository from a connection if it supports it.
func (registry *Registry) GetTransactionalRepository(name string) (TransactionalRepository, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	conn := registry.connections[name]
	if conn == nil {
		return nil, fmt.Errorf("%w (name=%q)", ErrConnectionNotFound, name)
	}

	// Check if the connection supports transactional behavior
	behaviors := conn.GetBehaviors()
	if !slices.Contains(behaviors, ConnectionBehaviorTransactional) {
		return nil, fmt.Errorf("%w (name=%q, operation=%q)",
			ErrConnectionNotSupported, name, "transactional operations")
	}

	// Try to get the repository from the raw connection
	repo, ok := conn.GetRawConnection().(TransactionalRepository)
	if !ok {
		return nil, fmt.Errorf("%w (name=%q, interface=%q)",
			ErrInterfaceNotImplemented, name, "TransactionalRepository")
	}

	return repo, nil
}

// GetQueryRepository returns a QueryRepository from a connection if it supports it.
func (registry *Registry) GetQueryRepository(name string) (QueryRepository, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	conn := registry.connections[name]
	if conn == nil {
		return nil, fmt.Errorf("%w (name=%q)", ErrConnectionNotFound, name)
	}

	// Check if the connection supports relational behavior
	behaviors := conn.GetBehaviors()
	if !slices.Contains(behaviors, ConnectionBehaviorRelational) {
		return nil, fmt.Errorf("%w (name=%q, operation=%q)",
			ErrConnectionNotSupported, name, "query operations")
	}

	// Try to get the repository from the raw connection
	repo, ok := conn.GetRawConnection().(QueryRepository)
	if !ok {
		return nil, fmt.Errorf("%w (name=%q, interface=%q)",
			ErrInterfaceNotImplemented, name, "QueryRepository")
	}

	return repo, nil
}

// GetCacheRepository returns a CacheRepository from a connection if it supports it.
func (registry *Registry) GetCacheRepository(name string) (CacheRepository, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	conn := registry.connections[name]
	if conn == nil {
		return nil, fmt.Errorf("%w (name=%q)", ErrConnectionNotFound, name)
	}

	// Check if the connection supports cache behavior
	behaviors := conn.GetBehaviors()
	if !slices.Contains(behaviors, ConnectionBehaviorCache) {
		return nil, fmt.Errorf("%w (name=%q, operation=%q)",
			ErrConnectionNotSupported, name, "cache operations")
	}

	// Try to get the cache repository from the raw connection
	repo, ok := conn.GetRawConnection().(CacheRepository)
	if !ok {
		return nil, fmt.Errorf("%w (name=%q, interface=%q)",
			ErrInterfaceNotImplemented, name, "CacheRepository")
	}

	return repo, nil
}

// GetQueueRepository returns a QueueRepository from a connection if it supports it.
func (registry *Registry) GetQueueRepository(name string) (QueueRepository, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	conn := registry.connections[name]
	if conn == nil {
		return nil, fmt.Errorf("%w (name=%q)", ErrConnectionNotFound, name)
	}

	// Check if the connection supports queue behavior
	behaviors := conn.GetBehaviors()
	if !slices.Contains(behaviors, ConnectionBehaviorQueue) {
		return nil, fmt.Errorf("%w (name=%q, operation=%q)",
			ErrConnectionNotSupported, name, "queue operations")
	}

	// Try to get the queue repository from the raw connection
	repo, ok := conn.GetRawConnection().(QueueRepository)
	if !ok {
		return nil, fmt.Errorf("%w (name=%q, interface=%q)",
			ErrInterfaceNotImplemented, name, "QueueRepository")
	}

	return repo, nil
}
