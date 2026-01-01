package postgres

import (
	"context"
	"fmt"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/database/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgxUnitOfWork implements output.UnitOfWork using pgx.Tx.
type pgxUnitOfWork struct {
	tx pgx.Tx
}

func (u *pgxUnitOfWork) Begin(ctx context.Context) (output.UnitOfWork, error) {
	// Already in a transaction, but we could support nested if needed.
	// For now, let's just return an error or the same one.
	return nil, fmt.Errorf("transaction already started")
}

func (u *pgxUnitOfWork) Commit(ctx context.Context) error {
	return u.tx.Commit(ctx)
}

func (u *pgxUnitOfWork) Rollback(ctx context.Context) error {
	return u.tx.Rollback(ctx)
}

func (u *pgxUnitOfWork) TaskRepository() output.TaskRepository {
	return repository.NewTaskRepository(u.tx)
}

func (u *pgxUnitOfWork) UserRepository() output.UserRepository {
	return repository.NewUserRepository(u.tx)
}

func (u *pgxUnitOfWork) RoleRepository() output.RoleRepository {
	return repository.NewRoleRepository(u.tx)
}

func (u *pgxUnitOfWork) PermissionRepository() output.PermissionRepository {
	return repository.NewPermissionRepository(u.tx)
}

func (u *pgxUnitOfWork) LabelRepository() output.LabelRepository {
	return repository.NewLabelRepository(u.tx)
}

func (u *pgxUnitOfWork) ACLRepository() output.ACLRepository {
	return repository.NewACLRepository(u.tx)
}

// pgxTransactionManager implements output.TransactionManager.
type pgxTransactionManager struct {
	pool *pgxpool.Pool
}

func NewTransactionManager(pool *pgxpool.Pool) output.TransactionManager {
	return &pgxTransactionManager{pool: pool}
}

func (m *pgxTransactionManager) RunInTransaction(ctx context.Context, fn func(uow output.UnitOfWork) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}

	uow := &pgxUnitOfWork{tx: tx}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(uow); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}
