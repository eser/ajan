package connfx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
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
	config *ConfigTarget,
) (Connection, error) {
	db, err := sql.Open(f.protocol, config.DSN)
	if err != nil {
		return nil, fmt.Errorf(
			"%w (protocol=%q, dsn=%q): %w",
			ErrFailedToOpenSQLConnection,
			f.protocol,
			config.DSN,
			err,
		)
	}

	// Configure connection pool if timeout is specified
	if config.Timeout > 0 {
		db.SetConnMaxLifetime(config.Timeout)
	}

	// Initial ping to verify connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close() // Ignore close error if ping fails

		return nil, fmt.Errorf("%w: %w", ErrFailedToPingSQL, err)
	}

	conn := &SQLConnection{
		protocol:   f.protocol,
		db:         db,
		state:      int32(ConnectionStateConnected),
		lastHealth: time.Time{},
	}

	return conn, nil
}

func (f *SQLConnectionFactory) GetProtocol() string {
	return f.protocol
}

func (f *SQLConnectionFactory) GetSupportedBehaviors() []ConnectionBehavior {
	return []ConnectionBehavior{ConnectionBehaviorStateful}
}

// Connection interface implementation

func (c *SQLConnection) GetBehaviors() []ConnectionBehavior {
	return []ConnectionBehavior{ConnectionBehaviorStateful}
}

func (c *SQLConnection) GetProtocol() string {
	return c.protocol
}

func (c *SQLConnection) GetState() ConnectionState {
	state := atomic.LoadInt32(&c.state)

	return ConnectionState(state)
}

func (c *SQLConnection) HealthCheck(ctx context.Context) *HealthStatus {
	start := time.Now()
	status := &HealthStatus{ //nolint:exhaustruct
		Timestamp: start,
	}

	// Ping the database
	err := c.db.PingContext(ctx)
	status.Latency = time.Since(start)

	if err != nil {
		atomic.StoreInt32(&c.state, int32(ConnectionStateError))
		status.State = ConnectionStateError
		status.Error = err
		status.Message = fmt.Sprintf("Health check failed: %v", err)

		return status
	}

	// Check connection stats
	stats := c.db.Stats()
	if stats.OpenConnections == 0 {
		atomic.StoreInt32(&c.state, int32(ConnectionStateDisconnected))
		status.State = ConnectionStateDisconnected
		status.Message = "No open connections"
	} else {
		atomic.StoreInt32(&c.state, int32(ConnectionStateConnected))
		status.State = ConnectionStateConnected
		status.Message = fmt.Sprintf("Connected (protocol=%s, open=%d, idle=%d)",
			c.protocol, stats.OpenConnections, stats.Idle)
	}

	c.lastHealth = start

	return status
}

func (c *SQLConnection) Close(ctx context.Context) error {
	atomic.StoreInt32(&c.state, int32(ConnectionStateDisconnected))

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
