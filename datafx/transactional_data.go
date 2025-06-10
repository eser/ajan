package datafx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/eser/ajan/connfx"
)

var (
	ErrTransactionNotSupported = errors.New("connection does not support transactions")
	ErrTransactionFailed       = errors.New("transaction failed")
)

// TransactionalData provides transactional data operations.
type TransactionalData struct {
	*Data
	transactionalRepo connfx.TransactionalRepository
}

// NewTransactional creates a new TransactionalData instance from a connfx connection.
// The connection must support transactional operations.
func NewTransactional(conn connfx.Connection) (*TransactionalData, error) {
	// First create a regular Data instance
	data, err := New(conn)
	if err != nil {
		return nil, err
	}

	// Check if the connection supports transactional behavior
	behaviors := conn.GetBehaviors()
	supportsTransactions := slices.Contains(behaviors, connfx.ConnectionBehaviorTransactional)

	if !supportsTransactions {
		return nil, fmt.Errorf(
			"%w: connection does not support transactional operations (protocol=%q)",
			ErrTransactionNotSupported,
			conn.GetProtocol(),
		)
	}

	// Get the transactional repository from the raw connection
	txRepo, ok := conn.GetRawConnection().(connfx.TransactionalRepository)
	if !ok {
		return nil, fmt.Errorf(
			"%w: connection does not implement TransactionalRepository interface (protocol=%q)",
			ErrTransactionNotSupported,
			conn.GetProtocol(),
		)
	}

	return &TransactionalData{
		Data:              data,
		transactionalRepo: txRepo,
	}, nil
}

// ExecuteTransaction executes a function within a transaction context.
// If the function returns an error, the transaction is rolled back.
// If the function succeeds, the transaction is committed.
func (td *TransactionalData) ExecuteTransaction(
	ctx context.Context,
	fn func(*TransactionData) error,
) error {
	// Begin transaction
	txCtx, err := td.transactionalRepo.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %w", ErrTransactionFailed, err)
	}

	// Create transaction-scoped data instance
	txData := &TransactionData{
		repository: txCtx.GetRepository(),
	}

	// Execute the function
	if err := fn(txData); err != nil {
		// Rollback on error
		if rollbackErr := txCtx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("%w: transaction failed with error %q and rollback failed: %w",
				ErrTransactionFailed, err.Error(), rollbackErr)
		}

		return fmt.Errorf("%w: %w", ErrTransactionFailed, err)
	}

	// Commit on success
	if err := txCtx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %w", ErrTransactionFailed, err)
	}

	return nil
}

// TransactionData provides data operations within a transaction context.
type TransactionData struct {
	repository connfx.DataRepository
}

// Get retrieves a value by key and unmarshals it into the provided destination.
func (td *TransactionData) Get(ctx context.Context, key string, dest any) error {
	data, err := td.repository.Get(ctx, key)
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
func (td *TransactionData) GetRaw(ctx context.Context, key string) ([]byte, error) {
	data, err := td.repository.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get raw key %q: %w", key, err)
	}

	if data == nil {
		return nil, fmt.Errorf("%w (key=%q)", ErrKeyNotFound, key)
	}

	return data, nil
}

// Set stores a value with the given key after marshaling it to JSON.
func (td *TransactionData) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToMarshal, key, err)
	}

	if err := td.repository.Set(ctx, key, data); err != nil {
		return fmt.Errorf("failed to set key %q: %w", key, err)
	}

	return nil
}

// SetRaw stores raw bytes with the given key.
func (td *TransactionData) SetRaw(ctx context.Context, key string, value []byte) error {
	if err := td.repository.Set(ctx, key, value); err != nil {
		return fmt.Errorf("failed to set raw key %q: %w", key, err)
	}

	return nil
}

// Update updates an existing value by key after marshaling it to JSON.
func (td *TransactionData) Update(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToMarshal, key, err)
	}

	if err := td.repository.Update(ctx, key, data); err != nil {
		return fmt.Errorf("failed to update key %q: %w", key, err)
	}

	return nil
}

// UpdateRaw updates an existing value with raw bytes by key.
func (td *TransactionData) UpdateRaw(ctx context.Context, key string, value []byte) error {
	if err := td.repository.Update(ctx, key, value); err != nil {
		return fmt.Errorf("failed to update raw key %q: %w", key, err)
	}

	return nil
}

// Remove deletes a value by key.
func (td *TransactionData) Remove(ctx context.Context, key string) error {
	if err := td.repository.Remove(ctx, key); err != nil {
		return fmt.Errorf("failed to remove key %q: %w", key, err)
	}

	return nil
}

// Exists checks if a key exists.
func (td *TransactionData) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := td.repository.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check if key %q exists: %w", key, err)
	}

	return exists, nil
}
