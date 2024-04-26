package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// UpsertElement upserts an element, no matter an entity or a relation
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

// LoadElementById by id returns the element with that id, if any
func (d *Dao) LoadElementById(ctx context.Context, id string) (patterns.Element, error) {
	if d == nil || d.pool == nil {
		return nil, errors.New("dao not initialized")
	}

	query := "select * from spat.loadelement($1)"
	rows, errRows := d.pool.Query(ctx, query, id)
	if errRows != nil {
		return nil, errRows
	} else {
		defer rows.Close()
	}

	// element may be a relation or an entity
	var relation patterns.Relation
	var entity patterns.Entity
	var isRelation bool
	var globalErr error
	counter := 0
	for rows.Next() {
		// read the raw values because some of them might be null
		var rawValues []any
		if raw, err := rows.Values(); err != nil {
			globalErr = errors.Join(globalErr, err)
			continue
		} else {
			rawValues = raw
		}

		// read the entier value at the first time, and fill values then
		if counter == 0 {
			// init values, read anything to build the data
			refType := int(rawValues[1].(int32))
			var traits []string
			var periodValue string
			if rawValues[4] != nil {
				periodValue = rawValues[4].(string)
			}

			// read the activity, note that period value may be null
			period, errPeriod := deserializePeriod(rawValues[3].(bool), periodValue)
			if errPeriod != nil {
				globalErr = errors.Join(globalErr, errPeriod)
				continue
			} else if period.IsEmptyPeriod() {
				// same period for all values
				break
			}

			if rawValues[2] != nil {
				anyTraits := rawValues[2].([]any)
				for _, trait := range anyTraits {
					traits = append(traits, trait.(string))
				}
			}

			if isEntityFromRefType(refType) {
				entity, _ = patterns.NewEntityWithId(id, traits, period)
			} else {
				relation = patterns.NewUnlinkedRelationWithId(id, traits)
				relation.SetActivePeriod(period)
			}
		}

		// even if we just built the object, keep going
		if isRelation {
			role := rawValues[6].(string)
			var roleValues []string
			if rawValues[7] != nil {
				rawRoles := rawValues[7].([]any)
				for _, rawRole := range rawRoles {
					roleValues = append(roleValues, rawRole.(string))
				}
			}

			errAdd := relation.SetValuesForRole(role, roleValues)
			if errAdd != nil {
				globalErr = errors.Join(globalErr, errAdd)
			}
		} else {
			var periodValue string
			if rawValues[10] != nil {
				periodValue = rawValues[10].(string)
			}

			attributePeriod, errAttributePeriod := deserializePeriod(rawValues[9].(bool), periodValue)
			if errAttributePeriod != nil {
				globalErr = errors.Join(globalErr, errAttributePeriod)
				continue
			} else if attributePeriod.IsEmptyPeriod() {
				continue
			}

			var attributeValue string
			if rawValues[8] != nil {
				attributeValue = rawValues[8].(string)
			}

			entity.AddValue(rawValues[7].(string), attributeValue, attributePeriod)
		}

		counter++
	}

	switch {
	case globalErr != nil:
		return nil, globalErr
	case isRelation && counter != 0:
		return &relation, globalErr
	case counter != 0:
		return &entity, globalErr
	default:
		return nil, globalErr
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

func (d *Dao) LoadElementRelationsCountAtMoment(ctx context.Context, id string, moment time.Time) ([]RelationalStatstDTO, error) {
	if d == nil || d.pool == nil {
		return nil, errors.New("dao not initialized")
	}

	query := "select * from spat.ElementRelationsCountAtMoment($1, $2)"
	rows, errRows := d.pool.Query(ctx, query, id, moment)
	if errRows != nil {
		return nil, errRows
	} else {
		defer rows.Close()
	}

	stats := make([]RelationalStatstDTO, 0)

	var globalErr error
	for rows.Next() {
		if rawValues, err := rows.Values(); err != nil {
			globalErr = errors.Join(globalErr, err)
			continue
		} else if rawValues[0] == nil {
			stats = append(stats, RelationalStatstDTO{
				Role:    rawValues[1].(string),
				Active:  rawValues[2].(bool),
				Counter: rawValues[3].(int64),
			})
		} else {
			stats = append(stats, RelationalStatstDTO{
				Trait:   rawValues[0].(string),
				Role:    rawValues[1].(string),
				Active:  rawValues[2].(bool),
				Counter: rawValues[3].(int64),
			})
		}
	}

	return stats, nil
}

func (d *Dao) LoadElementRelationsOperandsCountAtMoment(ctx context.Context, id string, moment time.Time) ([]RelationalOperandsStatstDTO, error) {
	if d == nil || d.pool == nil {
		return nil, errors.New("dao not initialized")
	}

	query := "select * from spat.ElementRelationsOperandsCountAtMoment($1, $2)"
	rows, errRows := d.pool.Query(ctx, query, id, moment)
	if errRows != nil {
		return nil, errRows
	} else {
		defer rows.Close()
	}

	stats := make([]RelationalOperandsStatstDTO, 0)

	var globalErr error
	for rows.Next() {
		if rawValues, err := rows.Values(); err != nil {
			globalErr = errors.Join(globalErr, err)
			continue
		} else if rawValues[0] == nil {
			rolesValues := make([]string, 0)
			values := rawValues[3].([]any)
			for _, value := range values {
				rolesValues = append(rolesValues, value.(string))
			}

			stats = append(stats, RelationalOperandsStatstDTO{
				Role:     rawValues[1].(string),
				Active:   rawValues[2].(bool),
				Operands: rolesValues,
				Counter:  rawValues[4].(int64),
			})
		} else {
			rolesValues := make([]string, 0)
			values := rawValues[3].([]any)
			for _, value := range values {
				rolesValues = append(rolesValues, value.(string))
			}

			stats = append(stats, RelationalOperandsStatstDTO{
				Trait:    rawValues[0].(string),
				Role:     rawValues[1].(string),
				Active:   rawValues[2].(bool),
				Operands: rolesValues,
				Counter:  rawValues[4].(int64),
			})
		}
	}

	return stats, nil
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

// deserializePeriod gets the values from the database and returns the matching period
func deserializePeriod(full bool, value string) (patterns.Period, error) {
	if full {
		return patterns.NewFullPeriod(), nil
	}

	values := strings.Split(value, "U")
	return patterns.DeserializePeriod(values, DATE_STORAGE_FORMAT)
}

// isEntityFromRefType returns true if value matches either entity or mixed
func isEntityFromRefType(value int) bool {
	return value == 1 || value == 10
}
