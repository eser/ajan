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
	ErrRepositoryOperation    = errors.New("repository operation failed")
)

// Store provides high-level data persistence operations.
type Store struct {
	conn       connfx.Connection
	repository connfx.Repository
}

// New creates a new Store instance from a connfx connection.
// The connection must support data repository operations.
func NewStore(conn connfx.Connection) (*Store, error) {
	if conn == nil {
		return nil, fmt.Errorf("%w: connection is nil", ErrConnectionNotSupported)
	}

	// Check if the connection supports data operations
	capabilities := conn.GetCapabilities()
	supportsStore := false

	for _, capability := range capabilities {
		if capability == connfx.ConnectionCapabilityKeyValue ||
			capability == connfx.ConnectionCapabilityDocument ||
			capability == connfx.ConnectionCapabilityRelational {
			supportsStore = true

			break
		}
	}

	if !supportsStore {
		return nil, fmt.Errorf("%w: connection does not support store operations (protocol=%q)",
			ErrConnectionNotSupported, conn.GetProtocol())
	}

	// Get the repository from the raw connection
	repo, ok := conn.GetRawConnection().(connfx.Repository)
	if !ok {
		return nil, fmt.Errorf(
			"%w: connection does not implement Repository interface (protocol=%q)",
			ErrConnectionNotSupported,
			conn.GetProtocol(),
		)
	}

	return &Store{
		conn:       conn,
		repository: repo,
	}, nil
}

// Get retrieves a value by key and unmarshals it into the provided destination.
func (s *Store) Get(ctx context.Context, key string, dest any) error {
	data, err := s.repository.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("%w (operation=get, key=%q): %w", ErrRepositoryOperation, key, err)
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
func (s *Store) GetRaw(ctx context.Context, key string) ([]byte, error) {
	data, err := s.repository.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf(
			"%w (operation=get_raw, key=%q): %w",
			ErrRepositoryOperation,
			key,
			err,
		)
	}

	if data == nil {
		return nil, fmt.Errorf("%w (key=%q)", ErrKeyNotFound, key)
	}

	return data, nil
}

// Set stores a value with the given key after marshaling it to JSON.
func (s *Store) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToMarshal, key, err)
	}

	if err := s.repository.Set(ctx, key, data); err != nil {
		return fmt.Errorf("%w (operation=set, key=%q): %w", ErrRepositoryOperation, key, err)
	}

	return nil
}

// SetRaw stores raw bytes with the given key.
func (s *Store) SetRaw(ctx context.Context, key string, value []byte) error {
	if err := s.repository.Set(ctx, key, value); err != nil {
		return fmt.Errorf("%w (operation=set_raw, key=%q): %w", ErrRepositoryOperation, key, err)
	}

	return nil
}

// Update updates an existing value by key after marshaling it to JSON.
func (s *Store) Update(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToMarshal, key, err)
	}

	if err := s.repository.Update(ctx, key, data); err != nil {
		return fmt.Errorf("%w (operation=update, key=%q): %w", ErrRepositoryOperation, key, err)
	}

	return nil
}

// UpdateRaw updates an existing value with raw bytes by key.
func (s *Store) UpdateRaw(ctx context.Context, key string, value []byte) error {
	if err := s.repository.Update(ctx, key, value); err != nil {
		return fmt.Errorf("%w (operation=update_raw, key=%q): %w", ErrRepositoryOperation, key, err)
	}

	return nil
}

// Remove deletes a value by key.
func (s *Store) Remove(ctx context.Context, key string) error {
	if err := s.repository.Remove(ctx, key); err != nil {
		return fmt.Errorf("%w (operation=remove, key=%q): %w", ErrRepositoryOperation, key, err)
	}

	return nil
}

// Exists checks if a key exists.
func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := s.repository.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf(
			"%w (operation=exists, key=%q): %w",
			ErrRepositoryOperation,
			key,
			err,
		)
	}

	return exists, nil
}

// GetConnection returns the underlying connfx connection.
func (s *Store) GetConnection() connfx.Connection {
	return s.conn
}

// GetRepository returns the underlying data repository.
func (s *Store) GetRepository() connfx.Repository {
	return s.repository
}
