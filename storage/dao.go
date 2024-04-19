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
	// DATE_STORAGE_FORMAT is golang representaion of dates. In terms of postgresql, it means YYYY-MM-DD HH24:MI:ss
	DATE_STORAGE_FORMAT = "2006-01-02T15:04:05"
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

func (d *Dao) UpsertElement(ctx context.Context, e patterns.Element) error {
	if entity, okEntity := e.(patterns.FormalInstance); okEntity {
		return d.UpsertEntity(ctx, entity)
	} else if relation, okRelation := e.(patterns.FormalRelation); okRelation {
		return d.UpsertRelation(ctx, relation)
	} else {
		return errors.New("unsupported element type")
	}
}

// UpsertEntity upserts an entity
func (d *Dao) UpsertEntity(ctx context.Context, e patterns.FormalInstance) error {
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
			index++
		}

		if _, err := d.pool.Exec(ctx, "call spat.addattributevaluesinentity($1,$2,$3,$4)", e.Id(), attr, values, periods); err != nil {
			globalErr = errors.Join(globalErr, errRead)
		}
	}

	if globalErr != nil {
		tx.Rollback(ctx)
		return globalErr
	} else if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("cannot commit transaction: %s", err.Error())
	} else {
		return nil
	}
}

// UpsertRelation upserts all the relation (traits, roles)
func (d *Dao) UpsertRelation(ctx context.Context, r patterns.FormalRelation) error {
	if d == nil || d.pool == nil {
		return errors.New("dao not initialized")
	}

	tx, errTx := d.pool.Begin(ctx)
	if errTx != nil {
		return fmt.Errorf("cannot start transaction: %s", errTx.Error())
	}

	var globalErr error

	if _, err := d.pool.Exec(ctx, "call spat.upsertrelation($1,$2,$3)", r.Id(), serializePeriod(r.ActivePeriod()), r.Traits()); err != nil {
		globalErr = errors.Join(globalErr, err)
	}

	if _, err := d.pool.Exec(ctx, "call spat.clearrolesforrelation($1)", r.Id()); err != nil {
		globalErr = errors.Join(globalErr, err)
	}

	// upsert each role
	for role, values := range r.GetValuesPerRole() {
		if _, err := d.pool.Exec(ctx, "call spat.upsertroleinrelation($1,$2,$3)", r.Id(), role, values); err != nil {
			globalErr = errors.Join(globalErr, err)
		}
	}

	if globalErr != nil {
		tx.Rollback(ctx)
		return globalErr
	} else if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("cannot commit transaction: %s", err.Error())
	} else {
		return nil
	}
}

// LoadActiveEntitiesAtTime returns active entity values at a given time
func (d *Dao) LoadActiveEntitiesAtTime(ctx context.Context, moment time.Time, trait string, valuesQuery map[string]string) ([]ElementDTO, error) {
	if d == nil || d.pool == nil {
		return nil, errors.New("dao not initialized")
	}

	query := queryForEntitiesAtDate(trait, valuesQuery)
	rows, errRows := d.pool.Query(ctx, query, moment)
	if errRows != nil {
		return nil, errRows
	} else {
		defer rows.Close()
	}

	activeValues := make(map[string]ElementDTO)

	var globalErr error
	for rows.Next() {
		var id, attribute, value string
		var traits []string
		hasAttribute := false

		// fill values, attribute and value may be null from the database.
		// If so, using a scan will raise an error.
		if rawValues, err := rows.Values(); err != nil {
			globalErr = errors.Join(globalErr, err)
			continue
		} else {
			id = rawValues[0].(string)
			// traits are read as []interface{}
			if rawValues[3] != nil {
				rawTraits := rawValues[3].([]any)
				for _, rawTrait := range rawTraits {
					if rawTrait == nil {
						continue
					}

					newTrait := rawTrait.(string)
					traits = append(traits, newTrait)
				}
			}

			// finally, read attribute
			if rawValues[1] == nil {
				hasAttribute = false
			} else {
				attribute = rawValues[1].(string)
				if rawValues[2] != nil {
					value = rawValues[2].(string)
				}
			}
		}

		// create it, may not add it
		newValue := EntityValueDTO{
			AttributeName:  attribute,
			AttributeValue: value,
		}

		if previous, found := activeValues[id]; found && hasAttribute {
			previous.Attributes = append(previous.Attributes, newValue)
			activeValues[id] = previous
		} else if !hasAttribute {
			newEntity := ElementDTO{
				Id:     id,
				Traits: traits,
			}

			activeValues[id] = newEntity
		} else {
			newEntity := ElementDTO{
				Id:         id,
				Traits:     traits,
				Attributes: []EntityValueDTO{newValue},
			}

			activeValues[id] = newEntity
		}
	}

	if globalErr != nil {
		return nil, globalErr
	}

	result := make([]ElementDTO, len(activeValues))
	index := 0

	for _, value := range activeValues {
		result[index] = value
		index++
	}

	return result, nil
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
	return t.UTC().Format(DATE_STORAGE_FORMAT)
}

// serializeInterval serializes a time interval
func serializeInterval(i patterns.Interval[time.Time]) string {
	return i.SerializeInterval(serializeTimestamp)
}
