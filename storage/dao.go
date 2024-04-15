package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zefrenchwan/patterns.git/patterns"
)

type Dao struct {
	pool *pgxpool.Pool
}

func NewDao(ctx context.Context, url string) (Dao, error) {
	var dao Dao
	if pool, errPool := pgxpool.New(ctx, url); errPool != nil {
		return dao, fmt.Errorf("dao creation failed: %s", errPool.Error())
	} else {
		dao.pool = pool
	}

	return dao, nil
}

func (d *Dao) UpsertEntity(ctx context.Context, e patterns.Entity) error {
	if d == nil || d.pool == nil {
		return errors.New("dao not initialized")
	}

	tx, errTx := d.pool.Begin(ctx)
	if errTx != nil {
		return errors.New("cannot start transaction")
	}

	d.pool.Exec()

	if err := tx.Commit(ctx); err != nil {
		return errors.New("cannot commit transaction")
	}

	return nil
}

func (d *Dao) UpsertRelation(ctx context.Context, r patterns.Relation) error {
	return nil
}

func (d *Dao) Close() {
	if d != nil && d.pool != nil {
		d.pool.Close()
	}
}

func serializePeriod(p patterns.Period) string {
	p.AsIntervals()
}

func deserializePeriod(p string) patterns.Period {

}
