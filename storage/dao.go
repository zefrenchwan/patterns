package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zefrenchwan/patterns.git/nodes"
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

// CheckUser returns true if login and password match
func (d *Dao) CheckUser(ctx context.Context, login, password string) (bool, error) {
	if d == nil || d.pool == nil {
		return false, errors.New("nil value")
	}

	var rows pgx.Rows
	if r, err := d.pool.Query(ctx, "select susers.test_user_password($1, $2)", login, password); err != nil {
		return false, err
	} else {
		rows = r
	}

	defer rows.Close()

	rows.Next()
	var result bool
	if err := rows.Scan(&result); err != nil {
		return false, err
	}

	return result, nil
}

// FindSecretForActiveUser returns the secret for an active user
func (d *Dao) FindSecretForActiveUser(ctx context.Context, login string) (string, error) {
	if d == nil || d.pool == nil {
		return "", errors.New("nil value")
	}

	var rows pgx.Rows
	if r, err := d.pool.Query(ctx, "select susers.find_secret_for_user($1)", login); err != nil {
		return "", err
	} else {
		rows = r
	}

	defer rows.Close()

	rows.Next()
	var result string
	if err := rows.Scan(&result); err != nil {
		return result, err
	}

	return result, nil
}

// UpsertUser changes user authentication if it exists, or insert user
func (d *Dao) UpsertUser(ctx context.Context, creator, login, password string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	_, errExec := d.pool.Exec(ctx, "call susers.upsert_user($1,$2,$3)", creator, login, password)
	return errExec
}

// CreateGraph returns the id of built graph, or an error.
func (d *Dao) CreateGraph(ctx context.Context, creator, name, description string, sources []string) (string, error) {
	if d == nil || d.pool == nil {
		return "", errors.New("nil value")
	}

	var errExec error
	newId := uuid.NewString()
	if len(sources) != 0 {
		_, errExec = d.pool.Exec(ctx,
			"call susers.create_graph_from_imports($1,$2,$3,$4,$5)",
			creator, newId, name, description, sources)
	} else {
		_, errExec = d.pool.Exec(ctx,
			"call susers.create_graph_from_scratch($1,$2,$3,$4)",
			creator, newId, name, description,
		)
	}

	return newId, errExec
}

// UpsertMetadataForGraph clears metadata and forces new values
func (d *Dao) UpsertMetadataForGraph(ctx context.Context, creator string, graphId string, metadata map[string][]string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	_, errExec := d.pool.Exec(ctx, "call susers.clear_graph_metadata($1, $2)", creator, graphId)
	if errExec != nil || len(metadata) == 0 {
		return errExec
	}

	for key, values := range metadata {
		_, errExec := d.pool.Exec(ctx, "call susers.upsert_graph_metadata_entry($1, $2, $3, $4)", creator, graphId, key, values)
		if errExec != nil || len(metadata) == 0 {
			return errExec
		}
	}

	return nil
}

// ListGraphsForUser returns the graphs an user has access to
func (d *Dao) ListGraphsForUser(ctx context.Context, user string) ([]GraphsForUserDTO, error) {
	var result []GraphsForUserDTO
	if d == nil || d.pool == nil {
		return result, errors.New("nil value")
	}

	rows, errLoad := d.pool.Query(ctx, "select * from susers.list_graphs_for_user($1) order by graph_id asc", user)
	if errLoad != nil {
		return result, errLoad
	}

	defer rows.Close()
	var currentData GraphsForUserDTO
	inserted := false
	var globalErr error

	for rows.Next() {
		inserted = false
		// expecting
		//  graph_id text, graph_roles text[],
		// graph_name text, graph_description text,
		// graph_md_key text, graph_md_values text[]
		var rawData []any
		if raw, err := rows.Values(); err != nil {
			globalErr = errors.Join(globalErr, err)
		} else {
			rawData = raw
		}

		graphId := rawData[0].(string)
		if currentData.Id != "" && graphId != currentData.Id {
			result = append(result, currentData)
			currentData = GraphsForUserDTO{}
			inserted = true
		}

		currentData.Id = graphId
		currentData.Name = rawData[2].(string)
		currentData.Roles = mapAnyToStringSlice(rawData[1])
		if rawData[3] != nil {
			currentData.Description = rawData[3].(string)
		}

		if rawData[4] == nil {
			continue
		} else if currentData.Metadata == nil {
			currentData.Metadata = make(map[string][]string)
		}

		key := rawData[4].(string)
		currentData.Metadata[key] = nil

		if rawData[5] == nil {
			continue
		}

		currentData.Metadata[key] = mapAnyToStringSlice(rawData[5])
	}

	if currentData.Id != "" && !inserted {
		result = append(result, currentData)
	}

	return result, globalErr
}

// UpsertElement adds an element to a given graph
func (d *Dao) UpsertElement(ctx context.Context, user string, graphId string, element nodes.Element) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	} else if element == nil {
		return nil
	}

	var elementType int
	var entity nodes.FormalInstance
	var relation nodes.FormalRelation
	switch newEntity, matchEntity := element.(nodes.FormalInstance); matchEntity {
	case true:
		elementType = 1
		entity = newEntity
	case false:
		elementType = 2
		relation = element.(nodes.FormalRelation)
	}

	_, errUpsertElement := d.pool.Exec(ctx,
		"call susers.upsert_element_in_graph($1, $2, $3, $4, $5, $6)",
		user, graphId, element.Id(), elementType,
		serializePeriod(element.ActivePeriod()),
		element.Traits(),
	)

	if errUpsertElement != nil {
		return errUpsertElement
	}

	var globalErr error
	if entity != nil {
		attributes := entity.Attributes()
		for _, attr := range attributes {
			values, errLoad := entity.PeriodValuesForAttribute(attr)
			if errLoad != nil {
				globalErr = errors.Join(globalErr, errLoad)
			}

			size := len(values)
			if size == 0 {
				continue
			}

			mappedValues := make([]string, size)
			mappedPeriods := make([]string, size)
			index := 0
			for value, period := range values {
				mappedValues[index] = value
				mappedPeriods[index] = serializePeriod(period)
			}

			//susers.upsert_attributes(p_user_login text, p_id text, p_name text, p_values text[], p_periods text[])
			_, errAttr := d.pool.Exec(ctx,
				"call susers.upsert_attributes($1, $2, $3, $4, $5)",
				user, entity.Id(), attr, mappedValues, mappedPeriods,
			)

			if errAttr != nil {
				globalErr = errors.Join(globalErr, errAttr)
			}
		}
	}

	if relation != nil {
		return nil
	}

	return globalErr
}

// Close closes the dao and the underlying pool
func (d *Dao) Close() {
	if d != nil && d.pool != nil {
		d.pool.Close()
	}
}

// serializePeriod returns the period as a string
func serializePeriod(p nodes.Period) string {
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
func serializeInterval(i nodes.Interval[time.Time]) string {
	return i.SerializeInterval(serializeTimestamp)
}

// deserializePeriod gets the values from the database and returns the matching period
func deserializePeriod(value string) (nodes.Period, error) {
	if strings.Contains(value, "]-oo;+oo[") {
		return nodes.NewFullPeriod(), nil
	}

	values := strings.Split(value, "U")
	return nodes.DeserializePeriod(values, DATE_STORAGE_FORMAT)
}

// isEntityFromRefType returns true if value matches either entity or mixed
func isEntityFromRefType(value int) bool {
	return value == 1 || value == 10
}

// mapAnySliceToStringSlice gets a slice of values and maps it to a string slice
func mapAnyToStringSlice(values any) []string {
	var result []string
	if values == nil {
		return result
	}

	rawValues := values.([]any)
	if len(rawValues) == 0 {
		return result
	}

	for _, value := range rawValues {
		if value == nil {
			continue
		}

		result = append(result, value.(string))
	}

	return result
}
