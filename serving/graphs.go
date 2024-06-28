package serving

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/zefrenchwan/patterns.git/graphs"
	"github.com/zefrenchwan/patterns.git/nodes"
	"github.com/zefrenchwan/patterns.git/storage"
)

// GraphDataDTO is the input to create a graph
type GraphDataDTO struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Metadata    map[string][]string `json:"metadata,omitempty"`
	Sources     []string            `json:"sources,omitempty"`
}

// createGraphHandler creates a graph: name, description, and metadata
func createGraphHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	var input GraphDataDTO
	if body, err := io.ReadAll(r.Body); err != nil {
		return NewServiceInternalServerError(err.Error())
	} else if errM := json.Unmarshal(body, &input); errM != nil {
		return NewServiceInternalServerError(errM.Error())
	} else if len(input.Name) == 0 {
		return NewServiceHttpClientError("expecting graph name")
	}

	newId, errCreate := wrapper.Dao.CreateGraph(wrapper.Ctx, user, input.Name, input.Description, input.Metadata, input.Sources)
	if errCreate != nil {
		return BuildApiErrorFromStorageError(errCreate)
	} else if errResponse := json.NewEncoder(w).Encode(newId); errResponse != nil {
		return NewServiceInternalServerError(errResponse.Error())
	}

	return nil
}

// upsertElementInGraphHandler loads an element dto and then saves it to database
func deleteGraphHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	graphId := r.PathValue("graphId")
	if len(graphId) == 0 {
		return NewServiceHttpClientError("expecting graph id")
	}

	if err := wrapper.Dao.DeleteGraph(wrapper.Ctx, user, graphId); err != nil {
		return BuildApiErrorFromStorageError(err)
	}

	w.WriteHeader(200)
	return nil
}

// listGraphHandler displays graphs available to an user
func listGraphHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	availableGraphs, errLoad := wrapper.Dao.ListGraphsForUser(wrapper.Ctx, user)
	if errLoad != nil {
		return NewServiceInternalServerError(errLoad.Error())
	} else if err := json.NewEncoder(w).Encode(availableGraphs); err != nil {
		return NewServiceInternalServerError(err.Error())
	}

	return nil
}

// loadGraphHandler loads a graph by id if user has access to it
func loadGraphHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	graphId := r.PathValue("graphId")
	if len(graphId) == 0 {
		return NewServiceHttpClientError("expecting graph id")
	}

	var rawGraph graphs.Graph
	if raw, err := wrapper.Dao.LoadGraphForUser(wrapper.Ctx, user, graphId); err != nil {
		return BuildApiErrorFromStorageError(err)
	} else {
		rawGraph = raw
	}

	switch rawGraph.Id {
	case "":
		w.WriteHeader(404)
		return nil
	default:
		if dto, err := storage.SerializeFullGraph(&rawGraph, storage.SerializeElement); err != nil {
			return NewServiceInternalServerError(err.Error())
		} else if err := json.NewEncoder(w).Encode(dto); err != nil {
			return NewServiceInternalServerError(err.Error())
		}
	}

	return nil
}

// loadGraphSinceHandler is a partial graph load, from a moment to +oo
func loadGraphSinceHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	graphId := r.PathValue("graphId")
	if len(graphId) == 0 {
		return NewServiceHttpClientError("expecting graph id")
	}

	var activePeriod nodes.Period
	if momentStr := r.PathValue("moment"); len(momentStr) != len(URL_DATE_FORMAT) {
		return NewServiceHttpClientError("Invalid date parameter")
	} else if value, err := DeserializeTimeFromURL(momentStr); err != nil {
		return NewServiceHttpClientError(err.Error())
	} else {
		sinceInterval := nodes.NewRightInfiniteTimeInterval(value, true)
		activePeriod = nodes.NewPeriod(sinceInterval)
	}

	var rawGraph graphs.Graph
	if raw, err := wrapper.Dao.LoadGraphForUserDuringPeriod(wrapper.Ctx, user, graphId, activePeriod); err != nil {
		return BuildApiErrorFromStorageError(err)
	} else {
		rawGraph = raw
	}

	switch rawGraph.Id {
	case "":
		w.WriteHeader(404)
		return nil
	default:
		if dto, err := storage.SerializeFullGraph(&rawGraph, storage.SerializeElement); err != nil {
			return NewServiceInternalServerError(err.Error())
		} else if err := json.NewEncoder(w).Encode(dto); err != nil {
			return NewServiceInternalServerError(err.Error())
		}
	}

	return nil
}

// loadGraphBetweenHandler gets two dates and loads data active during said time
func loadGraphBetweenHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	graphId := r.PathValue("graphId")
	if len(graphId) == 0 {
		return NewServiceHttpClientError("expecting graph id")
	}

	var activePeriod nodes.Period
	if startStr := r.PathValue("start"); len(startStr) != len(URL_DATE_FORMAT) {
		return NewServiceHttpClientError("Invalid date parameter")
	} else if startValue, err := DeserializeTimeFromURL(startStr); err != nil {
		return NewServiceHttpClientError(err.Error())
	} else if endStr := r.PathValue("end"); len(startStr) != len(URL_DATE_FORMAT) {
		return NewServiceHttpClientError("Invalid date parameter")
	} else if endValue, err := DeserializeTimeFromURL(endStr); err != nil {
		return NewServiceHttpClientError(err.Error())
	} else if valuesInterval, err := nodes.NewFiniteTimeInterval(startValue, endValue, true, true); err != nil {
		return NewServiceHttpClientError(err.Error())
	} else {
		activePeriod = nodes.NewPeriod(valuesInterval)
	}

	var rawGraph graphs.Graph
	if raw, err := wrapper.Dao.LoadGraphForUserDuringPeriod(wrapper.Ctx, user, graphId, activePeriod); err != nil {
		return BuildApiErrorFromStorageError(err)
	} else {
		rawGraph = raw
	}

	switch rawGraph.Id {
	case "":
		w.WriteHeader(404)
		return nil
	default:
		if dto, err := storage.SerializeFullGraph(&rawGraph, storage.SerializeElement); err != nil {
			return NewServiceInternalServerError(err.Error())
		} else if err := json.NewEncoder(w).Encode(dto); err != nil {
			return NewServiceInternalServerError(err.Error())
		}
	}

	return nil
}

// snapshotGraphHandler serializes a graph with visible elements at given time
func snapshotGraphHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	graphId := r.PathValue("graphId")
	if len(graphId) == 0 {
		return NewServiceHttpClientError("expecting graph id")
	}

	var moment time.Time
	if momentStr := r.PathValue("moment"); len(momentStr) != len(URL_DATE_FORMAT) {
		return NewServiceHttpClientError("Invalid date parameter")
	} else if value, err := DeserializeTimeFromURL(momentStr); err != nil {
		return NewServiceHttpClientError(err.Error())
	} else {
		moment = value.UTC()
	}

	// no need to load the full time, just load around said moment
	startTime := moment.Truncate(24 * time.Hour)
	endTime := moment.UTC().AddDate(0, 0, 1).Truncate(24 * time.Hour)
	optimizationInterval, errOptim := nodes.NewFiniteTimeInterval(startTime, endTime, true, true)
	if errOptim != nil {
		return NewServiceInternalServerError(errOptim.Error())
	}

	var rawGraph graphs.Graph
	optimizationPeriod := nodes.NewPeriod(optimizationInterval)
	if raw, err := wrapper.Dao.LoadGraphForUserDuringPeriod(wrapper.Ctx, user, graphId, optimizationPeriod); err != nil {
		return BuildApiErrorFromStorageError(err)
	} else {
		rawGraph = raw
	}

	switch rawGraph.Id {
	case "":
		w.WriteHeader(404)
		return nil
	default:
		if dto, err := storage.SerializeFullGraph(&rawGraph, storage.SerializerAtMoment(moment)); err != nil {
			return NewServiceInternalServerError(err.Error())
		} else if err := json.NewEncoder(w).Encode(dto); err != nil {
			return NewServiceInternalServerError(err.Error())
		}
	}

	return nil
}

// clearGraphsHandler clears all data about graphs (for test database)
func clearGraphsHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	errClear := wrapper.Dao.ClearGraph(wrapper.Ctx, user)
	if errClear != nil {
		return BuildApiErrorFromStorageError(errClear)
	}

	w.WriteHeader(200)
	return nil
}
