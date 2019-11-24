package store

import (
	"context"

	"github.com/yansal/sql/nest"
)

func Transaction(ctx context.Context, db nest.Querier, f func(ctx context.Context, tx nest.Querier) error) error {
	// TODO: move to sql/nest

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := f(ctx, tx); err != nil {
		if err := tx.Rollback(); err != nil {
			// TODO: return rollback error wrapping the original error
			return err
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		if err := tx.Rollback(); err != nil {
			// TODO: return rollback error wrapping the original error
			return err
		}
	}

	return nil
}
