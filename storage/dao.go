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

	var result string
	if !rows.Next() {
		return result, nil
	} else if err := rows.Scan(&result); err != nil {
		return result, err
	} else {
		return result, nil
	}
}

// ListUserDataAndSupervisedUsers provides all visible data and supervised errors
func (d *Dao) ListUserDataAndSupervisedUsers(ctx context.Context, login string) ([]UserAuthsDTO, error) {
	if d == nil || d.pool == nil {
		return nil, errors.New("nil value")
	}

	var rows pgx.Rows
	if r, err := d.pool.Query(ctx, "select * from susers.list_user_data_and_supervised_user_data($1)", login); err != nil {
		return nil, err
	} else {
		rows = r
	}

	userAuthValues := make(map[string]UserAuthsDTO)
	defer rows.Close()
	var globalErr error
	for rows.Next() {
		values, errValues := rows.Values()
		if errValues != nil {
			globalErr = errors.Join(globalErr, errValues)
			continue
		}

		userId := values[0].(string)
		userLogin := values[1].(string)
		userActive := values[2].(bool)
		roleName := values[3].(string)
		className := values[4].(string)
		allResources := values[5].(bool)
		authResources := mapAnyToStringSlice(values[6])
		unauthResources := mapAnyToStringSlice(values[7])

		var currentUserDto UserAuthsDTO
		if previous, found := userAuthValues[userId]; !found {
			currentUserDto = UserAuthsDTO{
				UserId:                  userId,
				ActiveUser:              userActive,
				Login:                   userLogin,
				ClassRoleAuthorizations: make(map[string]map[string]AuthDTO),
			}
		} else {
			currentUserDto = previous
		}

		if currentUserDto.ClassRoleAuthorizations[className] == nil {
			currentUserDto.ClassRoleAuthorizations[className] = make(map[string]AuthDTO)
		}

		currentAuthDTO := AuthDTO{
			AllAuthorized: allResources,
		}

		if !allResources && len(authResources) != 0 {
			currentAuthDTO.AuthorizedResources = authResources
		}

		if !allResources && len(unauthResources) != 0 {
			currentAuthDTO.AuthorizedResources = unauthResources
		}

		currentUserDto.ClassRoleAuthorizations[className][roleName] = currentAuthDTO
		userAuthValues[userId] = currentUserDto
	}

	allValues := make([]UserAuthsDTO, len(userAuthValues))
	index := 0
	for _, value := range userAuthValues {
		allValues[index] = value
		index++
	}

	return allValues, globalErr
}

// UpsertUser changes user authentication if it exists, or insert user
func (d *Dao) UpsertUser(ctx context.Context, creator, login, password string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	_, errExec := d.pool.Exec(ctx, "call susers.upsert_user($1,$2,$3)", creator, login, password)
	return errExec
}

// LockUser flags user as inactive
func (d *Dao) LockUser(ctx context.Context, actor, userId string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	_, errExec := d.pool.Exec(ctx, "call susers.lock_user($1,$2)", actor, userId)
	return errExec
}

// UnlockUser flags user as active
func (d *Dao) UnlockUser(ctx context.Context, actor, userId string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	_, errExec := d.pool.Exec(ctx, "call susers.unlock_user($1,$2)", actor, userId)
	return errExec
}

// DeleteUser deletes any information about the user
func (d *Dao) DeleteUser(ctx context.Context, actor, login string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	_, errExec := d.pool.Exec(ctx, "call susers.delete_user($1,$2)", actor, login)
	return errExec
}

// grantAllResourcesTo grants a role and a class to another user
func (d *Dao) GrantAllResourcesTo(ctx context.Context, granter, login, role, class string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	_, errExec := d.pool.Exec(ctx, "call susers.grant_all_role_auth_to($1, $2, $3, $4)", granter, login, role, class)
	return errExec
}

func (d *Dao) GrantResourcesTo(ctx context.Context, granter, login, role, class, resource string) error {
	return nil
}

func (d *Dao) RevokeAllResourcesTo(ctx context.Context, granter, login, role, class string) error {
	return nil
}

func (d *Dao) RevokeResourcesTo(ctx context.Context, granter, login, role, class, resource string) error {
	return nil
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
	var globalErr error
	values := make(map[string]AuthGraphDTO)

	for rows.Next() {
		var rawData []any
		if raw, err := rows.Values(); err != nil {
			globalErr = errors.Join(globalErr, err)
		} else {
			rawData = raw
		}

		graphId := rawData[0].(string)

		currentData, found := values[graphId]
		if !found {
			currentData.Id = graphId
			currentData.Name = rawData[2].(string)
			currentData.Roles = mapAnyToStringSlice(rawData[1])
			if rawData[3] != nil {
				currentData.Description = rawData[3].(string)
			}
			currentData.Metadata = make(map[string][]string)
		}

		var key string
		if rawData[4] != nil {
			key = rawData[4].(string)
			currentData.Metadata[key] = mapAnyToStringSlice(rawData[5])
		}

		values[graphId] = currentData
	}

	for _, value := range values {
		result = append(result, value)
	}

	return result, globalErr
}

// DeleteElement deletes an element from an user. May raise error on auth
func (d *Dao) DeleteElement(ctx context.Context, user, elementId string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	_, errExec := d.pool.Exec(ctx, "call susers.delete_element($1, $2)", user, elementId)
	return errExec
}

// DeleteGraph deletes an element from an user. May raise error on auth
func (d *Dao) DeleteGraph(ctx context.Context, user, graphId string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	_, errExec := d.pool.Exec(ctx, "call susers.delete_graph($1, $2)", user, graphId)
	return errExec
}

// LoadElementForUser returns an element, if any, matching that id
func (d *Dao) LoadElementForUser(ctx context.Context, user string, elementId string) (nodes.Element, error) {
	if d == nil || d.pool == nil {
		return nil, errors.New("nil value")
	}

	rows, errLoad := d.pool.Query(ctx, "select * from susers.load_element_by_id($1, $2)", user, elementId)
	if errLoad != nil {
		return nil, errLoad
	}

	var entity nodes.FormalInstance
	var relation nodes.FormalRelation
	var elementType = -1

	var globalErr error
	for rows.Next() {

		var rawValues []any
		if rawData, err := rows.Values(); err != nil {
			globalErr = errors.Join(globalErr, err)
			continue
		} else {
			rawValues = rawData
		}

		id := rawValues[0].(string)
		traits := mapAnyToStringSlice(rawValues[1])
		activity, errActivity := deserializePeriod(rawValues[2].(string))
		if errActivity != nil {
			globalErr = errors.Join(globalErr, errActivity)
			continue
		}

		var roleName string
		switch rawValues[3] {
		case nil:
			if elementType < 0 {
				elementType = 1

			}
		default:
			roleName = rawValues[3].(string)
			if elementType < 0 {
				elementType = 2
			}
		}

		roleValues := mapAnyToStringSlice(rawValues[4])
		var rolePeriods []nodes.Period

		var attributeName string
		var attributeValues []string
		var attributePeriods []nodes.Period

		if elementType == 1 {
			attributeName = rawValues[6].(string)
			attributeValues = mapAnyToStringSlice(rawValues[7])
			rawPeriods := mapAnyToStringSlice(rawValues[8])
			for _, rawPeriod := range rawPeriods {
				if period, err := deserializePeriod(rawPeriod); err == nil {
					attributePeriods = append(attributePeriods, period)
				} else {
					globalErr = errors.Join(globalErr, err)
					continue
				}
			}

			if len(attributePeriods) != len(attributeValues) {
				globalErr = errors.Join(globalErr, errors.New("invalid attributes request: size mismatch"))
				break
			}
		} else if elementType == 2 {
			rawPeriods := mapAnyToStringSlice(rawValues[5])
			if rawPeriods == nil {
				globalErr = errors.Join(globalErr, errors.New("invalid value for period: cannot be null"))
				break
			} else if len(roleValues) == 0 {
				globalErr = errors.Join(globalErr, errors.New("invalid value: cannot be null"))
				break
			} else if len(roleValues) != len(rawPeriods) {
				globalErr = errors.Join(globalErr, errors.New("invalid values and periods: size mismatch"))
				break
			}

			for _, rawPeriod := range rawPeriods {
				rolePeriod, errPeriod := deserializePeriod(rawPeriod)
				if errPeriod != nil {
					globalErr = errors.Join(globalErr, errPeriod)
					continue
				} else {
					rolePeriods = append(rolePeriods, rolePeriod)
				}
			}
		}

		switch elementType {
		case 1:
			if entity == nil {
				if newEntity, errEntity := nodes.NewEntityWithId(id, traits, activity); errEntity != nil {
					return nil, errors.Join(globalErr, errEntity)
				} else {
					entity = &newEntity
				}
			}

			for index := 0; index < len(attributeValues); index++ {
				entity.AddValue(attributeName, attributeValues[index], attributePeriods[index])
			}
		case 2:
			if relation == nil {
				relationValue := nodes.NewRelationWithId(id, traits)
				relation = &relationValue
			}

			for index := 0; index < len(roleValues); index++ {
				relation.AddPeriodValueForRole(roleName, roleValues[index], rolePeriods[index])
			}

		default:
			return nil, errors.New("mixed types not implemented")
		}
	}

	if globalErr != nil {
		return nil, globalErr
	}

	switch elementType {
	case 1:
		return entity, nil
	case 2:
		return relation, nil
	default:
		return nil, errors.New("mixed types not implemented")
	}
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
	const queryEntities = "select * from susers.transitive_load_entities_in_graph($1, $2) order by element_id, attribute_key asc"
	rowsEntities, errEntities := d.pool.Query(ctx, queryEntities, user, graphId)
	if errEntities != nil {
		return empty, errEntities
	}

	for rowsEntities.Next() {
		// read data from current line
		var rawEntityAttr []any
		if rawLine, errAttr := rowsEntities.Values(); errAttr != nil {
			globalErr = errors.Join(globalErr, errAttr)
			continue
		} else {
			rawEntityAttr = rawLine
		}

		currentGraphId := rawEntityAttr[0].(string)
		currentGraphEditable := rawEntityAttr[1].(bool)
		elementId := rawEntityAttr[2].(string)
		activity := nodes.NewEmptyPeriod()
		if rawEntityAttr[3] != nil {
			if a, err := deserializePeriod(rawEntityAttr[3].(string)); err != nil {
				globalErr = errors.Join(globalErr, err)
				continue
			} else if a.IsEmptyPeriod() {
				globalErr = errors.Join(globalErr, errors.New("empty period for element"))
				continue
			} else {
				activity = a
			}
		}

		var traits []string
		if rawEntityAttr[4] != nil {
			traits = mapAnyToStringSlice(rawEntityAttr[4])
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
		var attributePeriodValues []string
		if rawEntityAttr[7] != nil {
			attributeKey = rawEntityAttr[7].(string)
		}

		if rawEntityAttr[8] != nil {
			attributeValues = mapAnyToStringSlice(rawEntityAttr[8])
		}

		if rawEntityAttr[9] != nil {
			attributePeriodValues = mapAnyToStringSlice(rawEntityAttr[9])
		}

		localEquivalenceClassGraph := make(map[string]string)
		size := len(equivalenceClassGraph)
		for index := 0; index < size; index++ {
			localEquivalenceClassGraph[equivalenceClass[index]] = equivalenceClassGraph[index]
		}

		periodsError := false
		sizePeriodValues := len(attributePeriodValues)
		attributePeriods := make([]nodes.Period, sizePeriodValues)
		for index, periodValue := range attributePeriodValues {
			if newPeriod, err := deserializePeriod(periodValue); err != nil {
				globalErr = errors.Join(globalErr, err)
				periodsError = true
			} else {
				attributePeriods[index] = newPeriod
			}
		}

		if periodsError {
			continue
		}

		result.AddToFormalInstance(currentGraphId, currentGraphEditable, localEquivalenceClassGraph,
			elementId, traits, activity, attributeKey, attributeValues, attributePeriods,
		)
	}

	if globalErr != nil {
		return empty, globalErr
	}

	// globalErr is nil, proceed to relations
	// STEP THREE: RELATIONS
	const queryRelations = "select * from susers.transitive_load_relations_in_graph($1, $2) order by element_id asc"
	rowsRelation, errRelation := d.pool.Query(ctx, queryRelations, user, graphId)
	if errRelation != nil {
		return empty, errRelation
	}

	for rowsRelation.Next() {
		// read data from current line
		var rawRelation []any
		if rawLine, errAttr := rowsRelation.Values(); errAttr != nil {
			globalErr = errors.Join(globalErr, errAttr)
			continue
		} else {
			rawRelation = rawLine
		}

		currentGraphId := rawRelation[0].(string)
		currentGraphEditable := rawRelation[1].(bool)
		elementId := rawRelation[2].(string)
		activity := nodes.NewEmptyPeriod()
		if rawRelation[3] != nil {
			if a, err := deserializePeriod(rawRelation[3].(string)); err != nil {
				globalErr = errors.Join(globalErr, err)
				continue
			} else if a.IsEmptyPeriod() {
				globalErr = errors.Join(globalErr, errors.New("empty period for element"))
				continue
			} else {
				activity = a
			}
		}

		var traits []string
		if rawRelation[4] != nil {
			traits = mapAnyToStringSlice(rawRelation[4])
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
		var rolePeriods []string

		if rawRelation[7] != nil {
			roleName = rawRelation[7].(string)
		}

		if rawRelation[8] != nil {
			roleValues = mapAnyToStringSlice(rawRelation[8])
		}

		if len(roleValues) == 0 {
			globalErr = errors.Join(globalErr, errors.New("no value for a role in relation"))
			continue
		}

		if rawRelation[9] != nil {
			rolePeriods = mapAnyToStringSlice(rawRelation[9])
		}

		if len(rolePeriods) == 0 {
			globalErr = errors.Join(globalErr, errors.New("no value for a role in relation"))
			continue
		} else if len(rolePeriods) != len(roleValues) {
			globalErr = errors.Join(globalErr, errors.New("relation values and periods mismatch"))
			continue
		}

		localEquivalenceClassGraph := make(map[string]string)
		for index := 0; index < len(equivalenceClassGraph); index++ {
			localEquivalenceClassGraph[equivalenceClass[index]] = equivalenceClassGraph[index]
		}

		for index := 0; index < len(roleValues); index++ {
			switch rolePeriod, errPeriod := deserializePeriod(rolePeriods[index]); errPeriod {
			case nil:
				result.AddToFormalRelation(currentGraphId, currentGraphEditable, localEquivalenceClassGraph,
					elementId, traits, activity, roleName, roleValues[index], rolePeriod)
			default:
				globalErr = errors.Join(globalErr, errPeriod)
			}
		}
	}

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
				index++
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
		for role, links := range relation.PeriodValuesPerRole() {
			// serialize values and periods as slices
			linkValues := make([]string, 0)
			periodValues := make([]string, 0)
			for link, period := range links {
				if period.IsEmptyPeriod() {
					continue
				}

				linkValues = append(linkValues, link)
				periodValues = append(periodValues, serializePeriod(period))
			}

			// then call procedure
			_, errUpdate := transaction.Exec(ctx,
				"call susers.upsert_links($1, $2, $3, $4, $5)",
				user, relation.Id(), role, linkValues, periodValues,
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

// ClearGraph clear the whole graphs schema
func (d *Dao) ClearGraph(ctx context.Context, user string) error {
	if d == nil || d.pool == nil {
		return errors.New("nil value")
	}

	_, errExec := d.pool.Exec(ctx, "call susers.clear_graphs($1)", user)
	return errExec
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
