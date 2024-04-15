package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zefrenchwan/patterns.git/patterns"
)

const (
	// DATE_SERDE_FORMAT is golang representaion of dates. In terms of postgresql, it means YYYY-MM-DD HH24:MI:ss
	DATE_SERDE_FORMAT = "2006-01-02T15:04:05"
)

// Dao defines all database operations
type Dao struct {
	// pool to deal with multiple connections
	pool *pgxpool.Pool
}

// NewDao builds a new dao to connect a database via its url
func NewDao(ctx context.Context, url string) (Dao, error) {
	var dao Dao
	if pool, errPool := pgxpool.New(ctx, url); errPool != nil {
		return dao, fmt.Errorf("dao creation failed: %s", errPool.Error())
	} else {
		dao.pool = pool
	}

	return dao, nil
}

// UpsertEntity upserts an entity
func (d *Dao) UpsertEntity(ctx context.Context, e patterns.Entity) error {
	if d == nil || d.pool == nil {
		return errors.New("dao not initialized")
	}

	tx, errTx := d.pool.Begin(ctx)
	if errTx != nil {
		return fmt.Errorf("cannot start transaction: %s", errTx.Error())
	}

	var globalErr error

	// upsert entity per se
	if _, err := d.pool.Exec(ctx, "call spat.upsertentity($1,$2,$3)", e.Id(), serializePeriod(e.ActivePeriod()), e.Traits()); err != nil {
		globalErr = errors.Join(globalErr, err)
	}

	// upsert each attribute
	for _, attr := range e.Attributes() {
		valuePeriodMap, errRead := e.PeriodValuesForAttribute(attr)
		if errRead != nil {
			globalErr = errors.Join(globalErr, errRead)
			continue
		}

		values := make([]string, len(valuePeriodMap))
		periods := make([]string, len(valuePeriodMap))
		index := 0
		for value, period := range valuePeriodMap {
			values[index] = value
			periods[index] = serializePeriod(period)
		}

		d.pool.Exec(ctx, "call spat.addattributevaluesinentity($1,$2,$3,$4)", e.Id(), attr, values, periods)
	}

	if globalErr != nil {
		tx.Rollback(ctx)
		return globalErr
	} else if err := tx.Commit(ctx); err != nil {
		return errors.New("cannot commit transaction")
	} else {
		return nil
	}
}

func (d *Dao) UpsertRelation(ctx context.Context, r patterns.Relation) error {
	return nil
}

// Close closes the dao and the underlying pool
func (d *Dao) Close() {
	if d != nil && d.pool != nil {
		d.pool.Close()
	}
}

// serializePeriod returns the period as a string
func serializePeriod(p patterns.Period) string {
	switch {
	case p.IsEmptyPeriod():
		return "];["
	case p.IsFullPeriod():
		return "]-oo;+oo["
	default:
		result := ""
		for index, interval := range p.AsIntervals() {
			if index >= 1 {
				result = result + "U"
			}

			result = result + serializeInterval(interval)
		}

		return result
	}
}

// serializeTimestamp gets time value and returns it at the plpgsql format
func serializeTimestamp(t time.Time) string {
	return t.UTC().Format(DATE_SERDE_FORMAT)
}

// serializeInterval serializes a time interval
func serializeInterval(i patterns.Interval[time.Time]) string {
	return i.SerializeInterval(serializeTimestamp)
}
