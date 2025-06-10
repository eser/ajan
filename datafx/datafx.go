package datafx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/eser/ajan/connfx"
)

var (
	ErrConnectionNotSupported = errors.New("connection does not support required operations")
	ErrKeyNotFound            = errors.New("key not found")
	ErrFailedToMarshal        = errors.New("failed to marshal data")
	ErrFailedToUnmarshal      = errors.New("failed to unmarshal data")
	ErrInvalidData            = errors.New("invalid data")
)

// Data provides high-level data persistence operations.
type Data struct {
	conn       connfx.Connection
	repository connfx.DataRepository
}

// New creates a new Data instance from a connfx connection.
// The connection must support data repository operations.
func New(conn connfx.Connection) (*Data, error) {
	if conn == nil {
		return nil, fmt.Errorf("%w: connection is nil", ErrConnectionNotSupported)
	}

	// Check if the connection supports data operations
	behaviors := conn.GetBehaviors()
	supportsData := false

	for _, behavior := range behaviors {
		if behavior == connfx.ConnectionBehaviorKeyValue ||
			behavior == connfx.ConnectionBehaviorDocument ||
			behavior == connfx.ConnectionBehaviorRelational {
			supportsData = true

			break
		}
	}

	if !supportsData {
		return nil, fmt.Errorf("%w: connection does not support data operations (protocol=%q)",
			ErrConnectionNotSupported, conn.GetProtocol())
	}

	// Get the data repository from the raw connection
	repo, ok := conn.GetRawConnection().(connfx.DataRepository)
	if !ok {
		return nil, fmt.Errorf(
			"%w: connection does not implement DataRepository interface (protocol=%q)",
			ErrConnectionNotSupported,
			conn.GetProtocol(),
		)
	}

	return &Data{
		conn:       conn,
		repository: repo,
	}, nil
}

// Get retrieves a value by key and unmarshals it into the provided destination.
func (d *Data) Get(ctx context.Context, key string, dest any) error {
	data, err := d.repository.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get key %q: %w", key, err)
	}

	if data == nil {
		return fmt.Errorf("%w (key=%q)", ErrKeyNotFound, key)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToUnmarshal, key, err)
	}

	return nil
}

// GetRaw retrieves raw bytes by key.
func (d *Data) GetRaw(ctx context.Context, key string) ([]byte, error) {
	data, err := d.repository.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get raw key %q: %w", key, err)
	}

	if data == nil {
		return nil, fmt.Errorf("%w (key=%q)", ErrKeyNotFound, key)
	}

	return data, nil
}

// Set stores a value with the given key after marshaling it to JSON.
func (d *Data) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToMarshal, key, err)
	}

	if err := d.repository.Set(ctx, key, data); err != nil {
		return fmt.Errorf("failed to set key %q: %w", key, err)
	}

	return nil
}

// SetRaw stores raw bytes with the given key.
func (d *Data) SetRaw(ctx context.Context, key string, value []byte) error {
	if err := d.repository.Set(ctx, key, value); err != nil {
		return fmt.Errorf("failed to set raw key %q: %w", key, err)
	}

	return nil
}

// Update updates an existing value by key after marshaling it to JSON.
func (d *Data) Update(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToMarshal, key, err)
	}

	if err := d.repository.Update(ctx, key, data); err != nil {
		return fmt.Errorf("failed to update key %q: %w", key, err)
	}

	return nil
}

// UpdateRaw updates an existing value with raw bytes by key.
func (d *Data) UpdateRaw(ctx context.Context, key string, value []byte) error {
	if err := d.repository.Update(ctx, key, value); err != nil {
		return fmt.Errorf("failed to update raw key %q: %w", key, err)
	}

	return nil
}

// Remove deletes a value by key.
func (d *Data) Remove(ctx context.Context, key string) error {
	if err := d.repository.Remove(ctx, key); err != nil {
		return fmt.Errorf("failed to remove key %q: %w", key, err)
	}

	return nil
}

// Exists checks if a key exists.
func (d *Data) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := d.repository.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check if key %q exists: %w", key, err)
	}

	return exists, nil
}

// GetConnection returns the underlying connfx connection.
func (d *Data) GetConnection() connfx.Connection {
	return d.conn
}

// GetRepository returns the underlying data repository.
func (d *Data) GetRepository() connfx.DataRepository {
	return d.repository
}
