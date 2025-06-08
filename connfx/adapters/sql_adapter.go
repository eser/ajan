package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/eser/ajan/connfx"
)

var (
	ErrFailedToOpenSQLConnection = errors.New("failed to open SQL connection")
	ErrFailedToPingSQL           = errors.New("failed to ping SQL database")
	ErrInvalidConfigTypeSQL      = errors.New("invalid config type for SQL connection")
	ErrUnsupportedSQLProtocol    = errors.New("unsupported SQL protocol")
	ErrFailedToCloseSQLDB        = errors.New("failed to close SQL database")
)

// SQLConnection represents a SQL database connection.
type SQLConnection struct {
	lastHealth time.Time
	db         *sql.DB
	name       string
	protocol   string
	state      int32 // atomic field for connection state
}

// SQLConnectionFactory creates SQL connections.
type SQLConnectionFactory struct {
	protocol string
}

// NewSQLConnectionFactory creates a new SQL connection factory for a specific protocol.
func NewSQLConnectionFactory(protocol string) *SQLConnectionFactory {
	return &SQLConnectionFactory{
		protocol: protocol,
	}
}

func (f *SQLConnectionFactory) CreateConnection(
	ctx context.Context,
	config connfx.ConnectionConfig,
) (connfx.Connection, error) {
	// Build DSN from config
	dsn, err := f.buildDSN(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	db, err := sql.Open(f.protocol, dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToOpenSQLConnection, err)
	}

	// Configure connection pool if timeout is specified
	if baseConfig, ok := config.(*connfx.BaseConnectionConfig); ok {
		if baseConfig.Data.Timeout > 0 {
			db.SetConnMaxLifetime(baseConfig.Data.Timeout)
		}
	}

	// Initial ping to verify connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close() // Ignore close error if ping fails

		return nil, fmt.Errorf("%w: %w", ErrFailedToPingSQL, err)
	}

	conn := &SQLConnection{
		name:       config.GetName(),
		protocol:   f.protocol,
		db:         db,
		state:      int32(connfx.ConnectionStateConnected),
		lastHealth: time.Time{},
	}

	return conn, nil
}

func (f *SQLConnectionFactory) GetProtocol() string {
	return f.protocol
}

func (f *SQLConnectionFactory) GetSupportedBehaviors() []connfx.ConnectionBehavior {
	return []connfx.ConnectionBehavior{connfx.ConnectionBehaviorStateful}
}

func (f *SQLConnectionFactory) buildDSN(config connfx.ConnectionConfig) (string, error) {
	baseConfig, ok := config.(*connfx.BaseConnectionConfig)
	if !ok {
		return "", ErrInvalidConfigTypeSQL
	}

	data := baseConfig.Data

	// If DSN is provided directly, use it
	if data.DSN != "" {
		return data.DSN, nil
	}

	// Build DSN based on protocol
	switch f.protocol {
	case "postgres":
		return f.buildPostgresDSN(data), nil
	case "mysql":
		return f.buildMySQLDSN(data), nil
	case "sqlite":
		return f.buildSQLiteDSN(data), nil
	default:
		return "", fmt.Errorf("%w (protocol=%q)", ErrUnsupportedSQLProtocol, f.protocol)
	}
}

func (f *SQLConnectionFactory) buildPostgresDSN(data connfx.ConnectionConfigData) string {
	parts := []string{}
	if data.Host != "" {
		parts = append(parts, "host="+data.Host)
	}

	if data.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", data.Port))
	}

	if data.Database != "" {
		parts = append(parts, "dbname="+data.Database)
	}

	if data.Username != "" {
		parts = append(parts, "user="+data.Username)
	}

	if data.Password != "" {
		parts = append(parts, "password="+data.Password)
	}

	if !data.TLS {
		parts = append(parts, "sslmode=disable")
	}

	return strings.Join(parts, " ")
}

func (f *SQLConnectionFactory) buildMySQLDSN(data connfx.ConnectionConfigData) string {
	dsn := ""
	if data.Username != "" {
		dsn += data.Username
		if data.Password != "" {
			dsn += ":" + data.Password
		}

		dsn += "@"
	}

	if data.Host != "" {
		dsn += "tcp(" + data.Host
		if data.Port > 0 {
			dsn += fmt.Sprintf(":%d", data.Port)
		}

		dsn += ")"
	}

	if data.Database != "" {
		dsn += "/" + data.Database
	}

	return dsn
}

func (f *SQLConnectionFactory) buildSQLiteDSN(data connfx.ConnectionConfigData) string {
	if data.Database != "" {
		return data.Database
	}

	return ":memory:"
}

// Connection interface implementation

func (c *SQLConnection) GetName() string {
	return c.name
}

func (c *SQLConnection) GetBehaviors() []connfx.ConnectionBehavior {
	return []connfx.ConnectionBehavior{connfx.ConnectionBehaviorStateful}
}

func (c *SQLConnection) GetProtocol() string {
	return c.protocol
}

func (c *SQLConnection) GetState() connfx.ConnectionState {
	state := atomic.LoadInt32(&c.state)

	return connfx.ConnectionState(state)
}

func (c *SQLConnection) HealthCheck(ctx context.Context) *connfx.HealthStatus {
	start := time.Now()
	status := &connfx.HealthStatus{ //nolint:exhaustruct
		Timestamp: start,
	}

	// Ping the database
	err := c.db.PingContext(ctx)
	status.Latency = time.Since(start)

	if err != nil {
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateError))
		status.State = connfx.ConnectionStateError
		status.Error = err
		status.Message = fmt.Sprintf("Health check failed: %v", err)

		return status
	}

	// Check connection stats
	stats := c.db.Stats()
	if stats.OpenConnections == 0 {
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateDisconnected))
		status.State = connfx.ConnectionStateDisconnected
		status.Message = "No open connections"
	} else {
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateConnected))
		status.State = connfx.ConnectionStateConnected
		status.Message = fmt.Sprintf("Connected (protocol=%s, open=%d, idle=%d)",
			c.protocol, stats.OpenConnections, stats.Idle)
	}

	c.lastHealth = start

	return status
}

func (c *SQLConnection) Close(ctx context.Context) error {
	atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateDisconnected))

	if err := c.db.Close(); err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToCloseSQLDB, err)
	}

	return nil
}

func (c *SQLConnection) GetRawConnection() any {
	return c.db
}

// Additional SQL-specific methods

// GetDB returns the underlying *sql.DB instance.
func (c *SQLConnection) GetDB() *sql.DB {
	return c.db
}

// Stats returns database connection statistics.
func (c *SQLConnection) Stats() sql.DBStats {
	return c.db.Stats()
}

// Package level registration helpers

// RegisterPostgresAdapter registers the Postgres adapter with a registry.
func RegisterPostgresAdapter(registry *connfx.Registry) error {
	factory := NewSQLConnectionFactory("postgres")

	if err := registry.RegisterFactory(factory); err != nil {
		return fmt.Errorf("failed to register adapter (protocol=postgres): %w", err)
	}

	return nil
}

// RegisterMySQLAdapter registers the MySQL adapter with a registry.
func RegisterMySQLAdapter(registry *connfx.Registry) error {
	factory := NewSQLConnectionFactory("mysql")

	if err := registry.RegisterFactory(factory); err != nil {
		return fmt.Errorf("failed to register adapter (protocol=mysql): %w", err)
	}

	return nil
}

// RegisterSQLiteAdapter registers the SQLite adapter with a registry.
func RegisterSQLiteAdapter(registry *connfx.Registry) error {
	factory := NewSQLConnectionFactory("sqlite")

	if err := registry.RegisterFactory(factory); err != nil {
		return fmt.Errorf("failed to register adapter (protocol=sqlite): %w", err)
	}

	return nil
}
