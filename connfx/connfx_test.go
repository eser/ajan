package connfx_test

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/eser/ajan/connfx"
	"github.com/eser/ajan/connfx/adapters"
	"github.com/eser/ajan/logfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // Import SQLite driver
)

// Mock logger for testing.
func newMockLogger() *logfx.Logger {
	slogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ //nolint:exhaustruct
		Level: slog.LevelError, // Only show errors in tests
	}))

	return logfx.NewLoggerFromSlog(slogger)
}

func TestManager_SQLiteConnection(t *testing.T) {
	t.Parallel()

	logger := newMockLogger()
	manager := connfx.NewManager(logger)

	// Register SQLite adapter
	err := adapters.RegisterSQLiteAdapter(manager)
	require.NoError(t, err)

	ctx := t.Context()

	// Test SQLite connection (no external dependencies)
	config := connfx.NewConnectionConfig("test", connfx.ConnectionConfigData{ //nolint:exhaustruct
		Protocol: "sqlite",
		Database: ":memory:",
	})

	err = manager.AddConnection(ctx, config)
	require.NoError(t, err)

	// Get the connection
	conn, err := manager.GetConnection("test")
	require.NoError(t, err)
	assert.NotNil(t, conn)

	// Test connection properties
	assert.Equal(t, "test", conn.GetName())
	assert.Contains(t, conn.GetBehaviors(), connfx.ConnectionBehaviorStateful)
	assert.Equal(t, "sqlite", conn.GetProtocol())
	assert.Equal(t, connfx.ConnectionStateConnected, conn.GetState())

	// Test health check
	status := conn.HealthCheck(ctx)
	assert.Equal(t, connfx.ConnectionStateConnected, status.State)
	assert.NotZero(t, status.Timestamp)

	// Test type-safe connection extraction (should be *sql.DB)
	db, err := connfx.GetTypedConnection[*sql.DB](conn)
	require.NoError(t, err)
	assert.NotNil(t, db)

	// Verify it's actually a working *sql.DB
	err = db.Ping()
	require.NoError(t, err)

	// Close connection
	err = conn.Close(ctx)
	require.NoError(t, err)
}

func TestManager_LoadFromConfig(t *testing.T) {
	t.Parallel()

	logger := newMockLogger()
	manager := connfx.NewManager(logger)

	// Register SQLite adapter
	err := adapters.RegisterSQLiteAdapter(manager)
	require.NoError(t, err)

	ctx := t.Context()

	config := &connfx.Config{
		Connections: map[string]connfx.ConnectionConfigData{
			"default": { //nolint:exhaustruct
				Protocol: "sqlite",
				Database: ":memory:",
			},
		},
	}

	err = manager.LoadFromConfig(ctx, config)
	require.NoError(t, err)

	// Verify connection was loaded
	connections := manager.ListConnections()
	assert.Contains(t, connections, "default")

	// Get the connection
	conn, err := manager.GetConnection("default")
	require.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestManager_HealthCheck(t *testing.T) {
	t.Parallel()

	logger := newMockLogger()
	manager := connfx.NewManager(logger)

	// Register SQLite adapter
	err := adapters.RegisterSQLiteAdapter(manager)
	require.NoError(t, err)

	ctx := t.Context()

	// Add connection
	config := connfx.NewConnectionConfig(
		"sql_test",
		connfx.ConnectionConfigData{ //nolint:exhaustruct
			Protocol: "sqlite",
			Database: ":memory:",
		},
	)

	err = manager.AddConnection(ctx, config)
	require.NoError(t, err)

	// Test health check for all connections
	statuses := manager.HealthCheck(ctx)
	assert.Len(t, statuses, 1)
	assert.Contains(t, statuses, "sql_test")
	assert.Equal(t, connfx.ConnectionStateConnected, statuses["sql_test"].State)

	// Test named health check
	status, err := manager.HealthCheckNamed(ctx, "sql_test")
	require.NoError(t, err)
	assert.Equal(t, connfx.ConnectionStateConnected, status.State)

	// Test health check for non-existent connection
	_, err = manager.HealthCheckNamed(ctx, "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, connfx.ErrConnectionNotFound)
}

func TestManager_RemoveConnection(t *testing.T) {
	t.Parallel()

	logger := newMockLogger()
	manager := connfx.NewManager(logger)

	// Register SQLite adapter
	err := adapters.RegisterSQLiteAdapter(manager)
	require.NoError(t, err)

	ctx := t.Context()

	// Add connection
	config := connfx.NewConnectionConfig("test", connfx.ConnectionConfigData{ //nolint:exhaustruct
		Protocol: "sqlite",
		Database: ":memory:",
	})

	err = manager.AddConnection(ctx, config)
	require.NoError(t, err)

	// Verify connection exists
	connections := manager.ListConnections()
	assert.Contains(t, connections, "test")

	// Remove connection
	err = manager.RemoveConnection(ctx, "test")
	require.NoError(t, err)

	// Verify connection is removed
	connections = manager.ListConnections()
	assert.NotContains(t, connections, "test")

	// Try to get removed connection
	_, err = manager.GetConnection("test")
	assert.Error(t, err)
}

func TestManager_Close(t *testing.T) {
	t.Parallel()

	logger := newMockLogger()
	manager := connfx.NewManager(logger)

	// Register SQLite adapter
	err := adapters.RegisterSQLiteAdapter(manager)
	require.NoError(t, err)

	ctx := t.Context()

	// Add connection
	config := connfx.NewConnectionConfig("test", connfx.ConnectionConfigData{ //nolint:exhaustruct
		Protocol: "sqlite",
		Database: ":memory:",
	})

	err = manager.AddConnection(ctx, config)
	require.NoError(t, err)

	// Close all connections
	err = manager.Close(ctx)
	require.NoError(t, err)

	// Verify all connections are removed
	connections := manager.ListConnections()
	assert.Empty(t, connections)
}

func TestConnectionConfig_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  connfx.ConnectionConfig
		wantErr bool
	}{
		{
			name: "valid SQLite config",
			config: connfx.NewConnectionConfig(
				"test",
				connfx.ConnectionConfigData{ //nolint:exhaustruct
					Protocol: "sqlite",
					Database: "test.db",
				},
			),
			wantErr: false,
		},
		{
			name: "valid HTTP config",
			config: connfx.NewConnectionConfig(
				"test",
				connfx.ConnectionConfigData{ //nolint:exhaustruct
					Protocol: "http",
					URL:      "https://api.example.com",
				},
			),
			wantErr: false,
		},
		{
			name: "invalid config - no protocol",
			config: connfx.NewConnectionConfig(
				"test",
				connfx.ConnectionConfigData{}, //nolint:exhaustruct
			),
			wantErr: true,
		},
		{
			name: "invalid config - empty name",
			config: connfx.NewConnectionConfig("", connfx.ConnectionConfigData{ //nolint:exhaustruct
				Protocol: "sqlite",
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			baseConfig, ok := tt.config.(*connfx.BaseConnectionConfig)
			require.True(t, ok, "config should be of type *connfx.BaseConnectionConfig")

			err := baseConfig.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConnectionStates(t *testing.T) {
	t.Parallel()

	// Test that connection states are properly defined
	states := []connfx.ConnectionState{
		connfx.ConnectionStateUnknown,
		connfx.ConnectionStateConnected,
		connfx.ConnectionStateDisconnected,
		connfx.ConnectionStateError,
		connfx.ConnectionStateReconnecting,
	}

	for _, state := range states {
		assert.NotEmpty(t, state.String())
	}
}

func TestConnectionBehaviors(t *testing.T) {
	t.Parallel()

	// Test that connection behaviors are properly defined
	behaviors := []connfx.ConnectionBehavior{
		connfx.ConnectionBehaviorStateful,
		connfx.ConnectionBehaviorStateless,
		connfx.ConnectionBehaviorStreaming,
	}

	for _, behavior := range behaviors {
		assert.NotEmpty(t, string(behavior))
	}
}

func TestManager_BehaviorFiltering(t *testing.T) {
	t.Parallel()

	logger := newMockLogger()
	manager := connfx.NewManager(logger)

	// Register adapters
	err := adapters.RegisterSQLiteAdapter(manager)
	require.NoError(t, err)
	err = adapters.RegisterHTTPAdapter(manager)
	require.NoError(t, err)

	ctx := t.Context()

	// Add a stateful connection (SQLite)
	sqlConfig := connfx.NewConnectionConfig("db", connfx.ConnectionConfigData{ //nolint:exhaustruct
		Protocol: "sqlite",
		Database: ":memory:",
	})
	err = manager.AddConnection(ctx, sqlConfig)
	require.NoError(t, err)

	// Add a stateless connection (HTTP) - skip this test as it requires network
	// httpConfig := connfx.NewConnectionConfig("api", connfx.ConnectionConfigData{
	// 	Protocol: "http",
	// 	Behavior: "stateless",
	// 	URL:      "https://httpbin.org/status/200",
	// })
	// err = manager.AddConnection(ctx, httpConfig)
	// require.NoError(t, err)

	// Test behavior filtering
	statefulConnections := manager.GetStatefulConnections()
	assert.Len(t, statefulConnections, 1)
	assert.Equal(t, "db", statefulConnections[0].GetName())

	statelessConnections := manager.GetStatelessConnections()
	assert.Empty(t, statelessConnections) // No HTTP connection added

	// Test protocol filtering
	sqliteConnections := manager.GetConnectionsByProtocol("sqlite")
	assert.Len(t, sqliteConnections, 1)
	assert.Equal(t, "db", sqliteConnections[0].GetName())
}

func TestManager_AdapterRegistration(t *testing.T) {
	t.Parallel()

	logger := newMockLogger()
	manager := connfx.NewManager(logger)

	// Test registering adapters
	err := adapters.RegisterSQLiteAdapter(manager)
	require.NoError(t, err)

	err = adapters.RegisterHTTPAdapter(manager)
	require.NoError(t, err)

	// Test duplicate registration fails
	err = adapters.RegisterSQLiteAdapter(manager)
	require.Error(t, err)
	require.ErrorIs(t, err, connfx.ErrFactoryAlreadyRegistered)

	// Test listing protocols
	protocols := manager.ListRegisteredProtocols()
	assert.Contains(t, protocols, "sqlite")
	assert.Contains(t, protocols, "http")
}

func TestGetTypedConnection(t *testing.T) {
	t.Parallel()

	logger := newMockLogger()
	manager := connfx.NewManager(logger)

	// Register SQLite adapter
	err := adapters.RegisterSQLiteAdapter(manager)
	require.NoError(t, err)

	ctx := t.Context()

	// Add SQLite connection
	config := connfx.NewConnectionConfig("db", connfx.ConnectionConfigData{ //nolint:exhaustruct
		Protocol: "sqlite",
		Database: ":memory:",
	})
	err = manager.AddConnection(ctx, config)
	require.NoError(t, err)

	// Get connection
	conn, err := manager.GetConnection("db")
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Test successful type extraction
	db, err := connfx.GetTypedConnection[*sql.DB](conn)
	require.NoError(t, err)
	assert.NotNil(t, db)

	// Verify it's actually a working *sql.DB
	err = db.Ping()
	require.NoError(t, err)

	// Test failed type extraction (wrong type)
	_, err = connfx.GetTypedConnection[*http.Client](conn)
	require.Error(t, err)
	require.ErrorIs(t, err, connfx.ErrInvalidType)

	// Test with nil connection
	_, err = connfx.GetTypedConnection[*sql.DB](nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, connfx.ErrConnectionIsNil)
}
