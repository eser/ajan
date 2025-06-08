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
	ErrFactoryAlreadyRegistered = errors.New("factory already registered")
	ErrFailedToCloseConnections = errors.New("failed to close connections")
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
func (r *Registry) RegisterFactory(factory ConnectionFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	protocol := factory.GetProtocol()

	// Check if factory is already registered
	if _, exists := r.factories[protocol]; exists {
		return fmt.Errorf("%w (protocol=%q)", ErrFactoryAlreadyRegistered, protocol)
	}

	r.factories[protocol] = factory

	behaviors := factory.GetSupportedBehaviors()
	behaviorStrs := make([]string, len(behaviors))

	for i, b := range behaviors {
		behaviorStrs[i] = string(b)
	}

	r.logger.Info(
		"registered connection factory",
		slog.String("protocol", protocol),
		slog.Any("behaviors", behaviorStrs),
	)

	return nil
}

// GetDefault returns the default connection.
func (r *Registry) GetDefault() Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.connections[DefaultConnection]
}

// GetNamed returns a named connection.
func (r *Registry) GetNamed(name string) Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.connections[name]
}

// GetByBehavior returns all connections of a specific behavior.
func (r *Registry) GetByBehavior(behavior ConnectionBehavior) []Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var connections []Connection
	for _, conn := range r.connections {
		// Check if the connection supports the requested behavior
		if slices.Contains(conn.GetBehaviors(), behavior) {
			connections = append(connections, conn)
		}
	}

	return connections
}

// GetByProtocol returns all connections of a specific protocol.
func (r *Registry) GetByProtocol(protocol string) []Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var connections []Connection
	for _, conn := range r.connections {
		if conn.GetProtocol() == protocol {
			connections = append(connections, conn)
		}
	}

	return connections
}

// ListConnections returns all connection names.
func (r *Registry) ListConnections() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.connections))
	for name := range r.connections {
		names = append(names, name)
	}

	return names
}

// ListRegisteredProtocols returns all registered protocols.
func (r *Registry) ListRegisteredProtocols() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	protocols := make([]string, 0, len(r.factories))
	for protocol := range r.factories {
		protocols = append(protocols, protocol)
	}

	return protocols
}

// AddConnection adds a new connection to the registry.
func (r *Registry) AddConnection(ctx context.Context, config ConnectionConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := config.GetName()
	protocol := config.GetProtocol()

	// Check if connection already exists
	if _, exists := r.connections[name]; exists {
		return fmt.Errorf("%w (name=%q)", ErrConnectionAlreadyExists, name)
	}

	// Get factory for this protocol
	factory, exists := r.factories[protocol]
	if !exists {
		return fmt.Errorf("%w (protocol=%q)", ErrUnsupportedProtocol, protocol)
	}

	r.logger.Info(
		"creating connection",
		slog.String("name", name),
		slog.String("protocol", protocol),
	)

	// Create the connection
	conn, err := factory.CreateConnection(ctx, config)
	if err != nil {
		r.logger.Error(
			"failed to create connection",
			slog.String("error", err.Error()),
			slog.String("name", name),
			slog.String("protocol", protocol),
		)

		return fmt.Errorf("%w (name=%q): %w", ErrFailedToCreateConnection, name, err)
	}

	r.connections[name] = conn

	// Log the behaviors supported by this connection
	behaviors := conn.GetBehaviors()
	behaviorStrs := make([]string, len(behaviors))

	for i, b := range behaviors {
		behaviorStrs[i] = string(b)
	}

	r.logger.Info(
		"successfully added connection",
		slog.String("name", name),
		slog.String("protocol", protocol),
		slog.Any("behaviors", behaviorStrs),
	)

	return nil
}

// RemoveConnection removes a connection from the registry.
func (r *Registry) RemoveConnection(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, exists := r.connections[name]
	if !exists {
		return fmt.Errorf("%w (name=%q)", ErrConnectionNotFound, name)
	}

	// Close the connection
	if err := conn.Close(ctx); err != nil {
		r.logger.Warn(
			"error closing connection",
			slog.String("error", err.Error()),
			slog.String("name", name),
		)
	}

	delete(r.connections, name)

	r.logger.Info(
		"removed connection",
		slog.String("name", name),
	)

	return nil
}

// LoadFromConfig loads connections from configuration.
func (r *Registry) LoadFromConfig(ctx context.Context, config *Config) error {
	for name, connData := range config.Connections {
		nameLower := strings.ToLower(name)

		// Create connection config
		connConfig := NewConnectionConfig(nameLower, connData)

		// Validate configuration
		if err := connConfig.Validate(); err != nil {
			return fmt.Errorf("invalid config for connection %q: %w", nameLower, err)
		}

		// Add connection
		if err := r.AddConnection(ctx, connConfig); err != nil {
			return fmt.Errorf("failed to add connection %q: %w", nameLower, err)
		}
	}

	return nil
}

// HealthCheck performs health checks on all connections.
func (r *Registry) HealthCheck(ctx context.Context) map[string]*HealthStatus {
	r.mu.RLock()

	connections := make(map[string]Connection, len(r.connections))
	maps.Copy(connections, r.connections)
	r.mu.RUnlock()

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
func (r *Registry) HealthCheckNamed(ctx context.Context, name string) (*HealthStatus, error) {
	r.mu.RLock()
	conn, exists := r.connections[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w (name=%q)", ErrConnectionNotFound, name)
	}

	return conn.HealthCheck(ctx), nil
}

// Close closes all connections in the registry.
func (r *Registry) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []error

	for name, conn := range r.connections {
		if err := conn.Close(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to close connection %q: %w", name, err))
		}
	}

	// Clear the connections map
	r.connections = make(map[string]Connection)

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
