package datafx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type ContextKey string

const (
	ContextKeyUnitOfWork ContextKey = "unit-of-work"
)

var (
	ErrNoTransaction             = errors.New("no transaction in progress")
	ErrTransactionAlreadyStarted = errors.New("transaction already started")
	ErrTransactionBeginFailed    = errors.New("transaction begin failed")
	ErrTransactionCommitFailed   = errors.New("transaction commit failed")
	ErrTransactionRollbackFailed = errors.New("transaction rollback failed")
)

type TransactionStarter interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type TransactionController interface {
	Rollback() error
	Commit() error
}

type UnitOfWork struct {
	Context context.Context //nolint:containedctx

	TransactionStarter    TransactionStarter
	TransactionController TransactionController
}

func CurrentUnitOfWork(ctx context.Context) *UnitOfWork {
	uow, _ := ctx.Value(ContextKeyUnitOfWork).(*UnitOfWork)

	return uow
}

func NewUnitOfWork(transactionStarter TransactionStarter) *UnitOfWork {
	uow := &UnitOfWork{TransactionStarter: transactionStarter} //nolint:exhaustruct

	return uow
}

func (uow *UnitOfWork) Begin(ctx context.Context) error {
	if uow.TransactionController != nil {
		return ErrTransactionAlreadyStarted
	}

	opts := &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}

	newCtx := context.WithValue(ctx, ContextKeyUnitOfWork, uow)

	transaction, err := uow.TransactionStarter.BeginTx(newCtx, opts)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransactionBeginFailed, err)
	}

	uow.Context = newCtx
	uow.TransactionController = transaction

	return nil
}

func (uow *UnitOfWork) Commit() error {
	if uow.TransactionController == nil {
		return ErrNoTransaction
	}

	err := uow.TransactionController.Commit()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransactionCommitFailed, err)
	}

	uow.TransactionController = nil

	return nil
}

func (uow *UnitOfWork) Rollback() error {
	if uow.TransactionController == nil {
		return ErrNoTransaction
	}

	err := uow.TransactionController.Rollback()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransactionRollbackFailed, err)
	}

	uow.TransactionController = nil

	return nil
}

func (uow *UnitOfWork) Execute(ctx context.Context, fn func(uow *UnitOfWork) error) error {
	err := uow.Begin(ctx)
	if err != nil {
		return err
	}

	err = fn(uow)
	if err != nil {
		rollbackErr := uow.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}

		return err
	}

	return uow.Commit()
}
