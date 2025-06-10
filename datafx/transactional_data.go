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
	ErrTransactionOperation    = errors.New("transaction operation failed")
)

// TransactionalStore provides transactional store operations.
type TransactionalStore struct {
	*Store
	transactionalRepo connfx.TransactionalRepository
}

// NewTransactionalStore creates a new TransactionalStore instance from a connfx connection.
// The connection must support transactional operations.
func NewTransactionalStore(conn connfx.Connection) (*TransactionalStore, error) {
	// First create a regular Data instance
	store, err := NewStore(conn)
	if err != nil {
		return nil, err
	}

	// Check if the connection supports transactional behavior
	capabilities := conn.GetCapabilities()
	supportsTransactions := slices.Contains(capabilities, connfx.ConnectionCapabilityTransactional)

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

	return &TransactionalStore{
		Store:             store,
		transactionalRepo: txRepo,
	}, nil
}

// ExecuteTransaction executes a function within a transaction context.
// If the function returns an error, the transaction is rolled back.
// If the function succeeds, the transaction is committed.
func (ts *TransactionalStore) ExecuteTransaction(
	ctx context.Context,
	fn func(*TransactionStore) error,
) error {
	// Begin transaction
	txCtx, err := ts.transactionalRepo.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %w", ErrTransactionFailed, err)
	}

	// Create transaction-scoped data instance
	txData := &TransactionStore{
		Repository: txCtx.GetRepository(),
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

// TransactionStore provides data operations within a transaction context.
type TransactionStore struct {
	Repository connfx.Repository
}

// Get retrieves a value by key and unmarshals it into the provided destination.
func (ts *TransactionStore) Get(ctx context.Context, key string, dest any) error {
	data, err := ts.Repository.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("%w (operation=get, key=%q): %w", ErrTransactionOperation, key, err)
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
func (ts *TransactionStore) GetRaw(ctx context.Context, key string) ([]byte, error) {
	data, err := ts.Repository.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf(
			"%w (operation=get_raw, key=%q): %w",
			ErrTransactionOperation,
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
func (ts *TransactionStore) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToMarshal, key, err)
	}

	if err := ts.Repository.Set(ctx, key, data); err != nil {
		return fmt.Errorf("%w (operation=set, key=%q): %w", ErrTransactionOperation, key, err)
	}

	return nil
}

// SetRaw stores raw bytes with the given key.
func (ts *TransactionStore) SetRaw(ctx context.Context, key string, value []byte) error {
	if err := ts.Repository.Set(ctx, key, value); err != nil {
		return fmt.Errorf("%w (operation=set_raw, key=%q): %w", ErrTransactionOperation, key, err)
	}

	return nil
}

// Update updates an existing value by key after marshaling it to JSON.
func (ts *TransactionStore) Update(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w (key=%q): %w", ErrFailedToMarshal, key, err)
	}

	if err := ts.Repository.Update(ctx, key, data); err != nil {
		return fmt.Errorf("%w (operation=update, key=%q): %w", ErrTransactionOperation, key, err)
	}

	return nil
}

// UpdateRaw updates an existing value with raw bytes by key.
func (ts *TransactionStore) UpdateRaw(ctx context.Context, key string, value []byte) error {
	if err := ts.Repository.Update(ctx, key, value); err != nil {
		return fmt.Errorf(
			"%w (operation=update_raw, key=%q): %w",
			ErrTransactionOperation,
			key,
			err,
		)
	}

	return nil
}

// Remove deletes a value by key.
func (ts *TransactionStore) Remove(ctx context.Context, key string) error {
	if err := ts.Repository.Remove(ctx, key); err != nil {
		return fmt.Errorf("%w (operation=remove, key=%q): %w", ErrTransactionOperation, key, err)
	}

	return nil
}

// Exists checks if a key exists.
func (ts *TransactionStore) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := ts.Repository.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf(
			"%w (operation=exists, key=%q): %w",
			ErrTransactionOperation,
			key,
			err,
		)
	}

	return exists, nil
}
