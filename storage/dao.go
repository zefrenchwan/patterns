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
	"github.com/zefrenchwan/patterns.git/graphs"
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
func (d *Dao) CreateGraph(ctx context.Context, creator, name, description string, metadata map[string][]string, sources []string) (string, error) {
	if d == nil || d.pool == nil {
		return "", errors.New("nil value")
	}

	transaction, errTransaction := d.pool.Begin(ctx)
	if errTransaction != nil {
		errRollback := transaction.Rollback(ctx)
		return "", errors.Join(errTransaction, errRollback)
	}

	var errExec error
	newId := uuid.NewString()
	if len(sources) != 0 {
		_, errExec = transaction.Exec(ctx,
			"call susers.create_graph_from_imports($1,$2,$3,$4,$5)",
			creator, newId, name, description, sources)
	} else {
		_, errExec = transaction.Exec(ctx,
			"call susers.create_graph_from_scratch($1,$2,$3,$4)",
			creator, newId, name, description,
		)
	}

	if errExec != nil {
		errRollback := transaction.Rollback(ctx)
		return "", errors.Join(errExec, errRollback)
	}

	_, errExec = transaction.Exec(ctx, "call susers.clear_graph_metadata($1, $2)", creator, newId)
	if errExec != nil {
		errRollback := transaction.Rollback(ctx)
		return "", errors.Join(errExec, errRollback)
	}

	for key, values := range metadata {
		_, errExec := transaction.Exec(ctx, "call susers.upsert_graph_metadata_entry($1, $2, $3, $4)", creator, newId, key, values)
		if errExec != nil {
			transaction.Rollback(ctx)
			return "", errExec
		}
	}

	errCommit := transaction.Commit(ctx)
	return newId, errCommit
}

// UpsertMetadataForGraph clears metadata and forces new values
func (d *Dao) UpsertMetadataForGraph(ctx context.Context, creator string, graphId string, metadata map[string][]string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	transaction, errTransaction := d.pool.Begin(ctx)
	if errTransaction != nil {
		errRollback := transaction.Rollback(ctx)
		return errors.Join(errTransaction, errRollback)
	}

	_, errExec := transaction.Exec(ctx, "call susers.clear_graph_metadata($1, $2)", creator, graphId)
	if errExec != nil || len(metadata) == 0 {
		errRollback := transaction.Rollback(ctx)
		return errors.Join(errExec, errRollback)
	}

	for key, values := range metadata {
		_, errExec := d.pool.Exec(ctx, "call susers.upsert_graph_metadata_entry($1, $2, $3, $4)", creator, graphId, key, values)
		if errExec != nil || len(metadata) == 0 {
			errRollback := transaction.Rollback(ctx)
			return errors.Join(errExec, errRollback)
		}
	}

	errCommit := transaction.Commit(ctx)
	return errCommit
}

// ListGraphsForUser returns the graphs an user has access to
func (d *Dao) ListGraphsForUser(ctx context.Context, user string) ([]AuthGraphDTO, error) {
	var result []AuthGraphDTO
	if d == nil || d.pool == nil {
		return result, errors.New("nil value")
	}

	rows, errLoad := d.pool.Query(ctx, "select * from susers.list_graphs_for_user($1) order by graph_id asc", user)
	if errLoad != nil {
		return result, errLoad
	}

	defer rows.Close()
	var currentData AuthGraphDTO
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
			currentData = AuthGraphDTO{}
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

// LoadGraphForUser loads a graph and dependencies given base id for a given user
func (d *Dao) LoadGraphForUser(ctx context.Context, user string, graphId string) (graphs.Graph, error) {
	var empty graphs.Graph
	if d == nil || d.pool == nil {
		return empty, errors.New("nil value")
	}

	result := graphs.NewEmptyGraph()

	// STEP ONE: LOAD METADATA
	rows, errMetadata := d.pool.Query(ctx, "select * from susers.load_graph_metadata($1, $2)", user, graphId)
	if errMetadata != nil {
		return empty, errMetadata
	}

	var globalErr error
	for rows.Next() {
		result.Id = graphId

		var entryKey string
		var entryValues []string

		if rawValues, err := rows.Values(); err != nil {
			globalErr = errors.Join(globalErr, err)
			continue
		} else {
			result.Name = rawValues[0].(string)

			if rawValues[1] != nil {
				result.Description = rawValues[1].(string)
			}

			if rawValues[2] != nil {
				entryKey = rawValues[2].(string)
			}

			if rawValues[3] != nil {
				entryValues = mapAnyToStringSlice(rawValues[3])
			}
		}

		if entryKey != "" {
			if result.Metadata == nil {
				result.Metadata = make(map[string][]string)
			}

			result.Metadata[entryKey] = entryValues
		}
	}

	if globalErr != nil {
		return empty, globalErr
	}

	// globalErr is nil, proceed to entities
	// STEP TWO: ENTITIES

	// Database contract is:
	// susers.transitive_load_entities_in_graph(p_user_login text, p_id text)
	// returns table (
	// graph_id text, editable bool,
	// element_id text, activity text, traits text[],
	// equivalence_class text[], equivalence_class_graph text[],
	// attribute_key text, attribute_values text[], attribute_periods text[]
	const queryEntities = "select * from susers.transitive_load_entities_in_graph($1, $2) order by element_id asc"
	rows, errEntities := d.pool.Query(ctx, queryEntities, user, graphId)
	if errEntities != nil {
		return empty, errEntities
	}

	// due to the order by element_id, we may load elements one by one and not keep the map of partial loaded elements
	var previousId string
	var currentGraphId string
	var currentGraphEditable bool
	var currentEntity nodes.FormalInstance
	var localEquivalenceClassGraph map[string]string

	for rows.Next() {
		// read data from current line
		var rawEntityAttr []any
		if rawLine, errAttr := rows.Values(); errAttr != nil {
			globalErr = errors.Join(globalErr, errAttr)
			continue
		} else {
			rawEntityAttr = rawLine
		}

		currentGraphId = rawEntityAttr[0].(string)
		currentGraphEditable = rawEntityAttr[1].(bool)
		elementId := rawEntityAttr[2].(string)
		activity := nodes.NewEmptyPeriod()
		if rawEntityAttr[3] != nil {
			if a, err := deserializePeriod(rawEntityAttr[3].(string)); err != nil {
				globalErr = errors.Join(globalErr, err)
				previousId = elementId
				continue
			} else if a.IsEmptyPeriod() {
				globalErr = errors.Join(globalErr, errors.New("empty period for element"))
				previousId = elementId
				continue
			} else {
				activity = a
			}
		}

		var traits []string
		if rawEntityAttr[4] != nil {
			traits = mapAnyToStringSlice(rawEntityAttr[4])
		}

		if previousId == "" {
			newEntity, errCreate := nodes.NewEntityWithId(elementId, traits, activity)
			if errCreate != nil {
				globalErr = errors.Join(globalErr, errCreate)
				previousId = elementId
				continue
			} else {
				currentEntity = &newEntity
			}
		}

		var equivalenceClass []string
		var equivalenceClassGraph []string
		if rawEntityAttr[5] != nil {
			equivalenceClass = mapAnyToStringSlice(rawEntityAttr[5])
		}

		if rawEntityAttr[6] != nil {
			equivalenceClassGraph = mapAnyToStringSlice(rawEntityAttr[6])
		}

		var attributeKey string
		var attributeValues []string
		var attributePeriods []string
		if rawEntityAttr[7] != nil {
			attributeKey = rawEntityAttr[7].(string)
		}

		if rawEntityAttr[8] != nil {
			attributeValues = mapAnyToStringSlice(rawEntityAttr[8])
		}

		if rawEntityAttr[9] != nil {
			attributePeriods = mapAnyToStringSlice(rawEntityAttr[9])
		}

		localEquivalenceClassGraph = nil
		localEquivalenceClassGraph = make(map[string]string)
		size := len(equivalenceClassGraph)
		for index := 0; index < size; index++ {
			localEquivalenceClassGraph[equivalenceClass[index]] = equivalenceClassGraph[index]
		}

		// OK, now update the element
		if previousId != "" && elementId != previousId {
			// previous value is complete, now insert
			result.SetLoadedElement(currentEntity, currentGraphId, currentGraphEditable, localEquivalenceClassGraph)
			// make new element
			currentValue, _ := nodes.NewEntityWithId(elementId, traits, activity)
			currentEntity = &currentValue
		}

		if attributeKey == "" {
			continue
		}
		// add current information to current element
		attrSize := len(attributeValues)
		for index := 0; index < attrSize; index++ {
			period, errPeriod := deserializePeriod(attributePeriods[index])
			if errPeriod != nil {
				globalErr = errors.Join(globalErr, errPeriod)
				previousId = elementId
				continue
			}

			currentEntity.AddValue(attributeKey, attributeValues[index], period)
		}

		// then, set value for previousId just before going to previous element
		previousId = elementId
	}

	// set last value. Either it did not exist and is saved, or it is just resaved
	result.SetLoadedElement(currentEntity, currentGraphId, currentGraphEditable, localEquivalenceClassGraph)

	if globalErr != nil {
		return empty, globalErr
	}

	// globalErr is nil, proceed to relations
	// STEP THREE: RELATIONS
	localEquivalenceClassGraph = nil
	localEquivalenceClassGraph = make(map[string]string)
	previousId = ""
	// database contract is:
	// create or replace function susers.transitive_load_relations_in_graph(p_user_login text, p_id text)
	// returns table (
	// 	graph_id text, editable bool,
	// 	element_id text, activity text, traits text[],
	// 	equivalence_class text[], equivalence_class_graph text[],
	// 	role_in_relation text, role_values text[]
	// )

	var currentRelation nodes.FormalRelation
	const queryRelations = "select * from susers.transitive_load_relations_in_graph($1, $2) order by element_id asc"
	rows, errRelation := d.pool.Query(ctx, queryRelations, user, graphId)
	if errRelation != nil {
		return empty, errEntities
	}

	for rows.Next() {
		// read data from current line
		var rawRelation []any
		if rawLine, errAttr := rows.Values(); errAttr != nil {
			globalErr = errors.Join(globalErr, errAttr)
			continue
		} else {
			rawRelation = rawLine
		}

		currentGraphId = rawRelation[0].(string)
		currentGraphEditable = rawRelation[1].(bool)
		elementId := rawRelation[2].(string)
		activity := nodes.NewEmptyPeriod()
		if rawRelation[3] != nil {
			if a, err := deserializePeriod(rawRelation[3].(string)); err != nil {
				globalErr = errors.Join(globalErr, err)
				previousId = elementId
				continue
			} else if a.IsEmptyPeriod() {
				globalErr = errors.Join(globalErr, errors.New("empty period for element"))
				previousId = elementId
				continue
			} else {
				activity = a
			}
		}

		var traits []string
		if rawRelation[4] != nil {
			traits = mapAnyToStringSlice(rawRelation[4])
		}

		if previousId == "" {
			newRelation := nodes.NewUnlinkedRelationWithId(elementId, traits)
			currentRelation = &newRelation
			currentRelation.SetActivePeriod(activity)
		}

		var equivalenceClass []string
		var equivalenceClassGraph []string
		if rawRelation[5] != nil {
			equivalenceClass = mapAnyToStringSlice(rawRelation[5])
		}

		if rawRelation[6] != nil {
			equivalenceClassGraph = mapAnyToStringSlice(rawRelation[6])
		}

		var roleName string
		var roleValues []string
		if rawRelation[7] != nil {
			roleName = rawRelation[7].(string)
		}

		if rawRelation[8] != nil {
			roleValues = mapAnyToStringSlice(rawRelation[8])
		}

		if len(roleValues) == 0 {
			globalErr = errors.Join(globalErr, errors.New("no value for a role in relation"))
			previousId = elementId
			continue
		}

		currentRelation.SetValuesForRole(roleName, roleValues)

		localEquivalenceClassGraph = nil
		localEquivalenceClassGraph = make(map[string]string)
		size := len(equivalenceClassGraph)
		for index := 0; index < size; index++ {
			localEquivalenceClassGraph[equivalenceClass[index]] = equivalenceClassGraph[index]
		}

		// OK, now update the element
		if previousId != "" && elementId != previousId {
			// previous value is complete, now insert
			result.SetLoadedElement(currentRelation, currentGraphId, currentGraphEditable, localEquivalenceClassGraph)
			// make new element
			currentValue, _ := nodes.NewEntityWithId(elementId, traits, activity)
			currentEntity = &currentValue
		}

		// then, set value for previousId just before going to previous element
		previousId = elementId
	}

	// set last value. Either it did not exist and is saved, or it is just resaved
	result.SetLoadedElement(currentRelation, currentGraphId, currentGraphEditable, localEquivalenceClassGraph)

	if globalErr != nil {
		return empty, globalErr
	}

	return result, nil
}

// UpsertElement adds an element to a given graph
func (d *Dao) UpsertElement(ctx context.Context, user string, graphId string, element nodes.Element) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	} else if element == nil {
		return nil
	}

	transaction, errTransaction := d.pool.Begin(ctx)
	if errTransaction != nil {
		return errTransaction
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

	_, errUpsertElement := transaction.Exec(ctx,
		"call susers.upsert_element_in_graph($1, $2, $3, $4, $5, $6)",
		user, graphId, element.Id(), elementType,
		serializePeriod(element.ActivePeriod()),
		element.Traits(),
	)

	if errUpsertElement != nil {
		errRollback := transaction.Rollback(ctx)
		return errors.Join(errUpsertElement, errRollback)
	}

	// all checks performed before, so direct access to this function
	_, errClearElement := transaction.Exec(ctx,
		"call sgraphs.clear_element_data_in_dependent_tables($1)",
		element.Id(),
	)

	if errClearElement != nil {
		errRollback := transaction.Rollback(ctx)
		return errors.Join(errClearElement, errRollback)
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
			_, errAttr := transaction.Exec(ctx,
				"call susers.upsert_attributes($1, $2, $3, $4, $5)",
				user, entity.Id(), attr, mappedValues, mappedPeriods,
			)

			if errAttr != nil {
				globalErr = errors.Join(globalErr, errAttr)
			}
		}
	} else if relation != nil {
		for role, links := range relation.GetValuesPerRole() {
			_, errUpdate := transaction.Exec(ctx,
				"call susers.upsert_links($1, $2, $3, $4)",
				user, relation.Id(), role, links,
			)

			if errUpdate != nil {
				globalErr = errors.Join(globalErr, errUpdate)
			}
		}
	}

	if globalErr != nil {
		errRollback := transaction.Rollback(ctx)
		if errRollback != nil {
			globalErr = errors.Join(globalErr, errRollback)
		}

		return globalErr
	}

	errCommit := transaction.Commit(ctx)
	return errCommit
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
