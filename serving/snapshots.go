package serving

import (
	"encoding/json"
	"net/http"
	"time"
)

const (
	// DATE_WS_FORMAT is the format for moments
	DATE_WS_FORMAT = "2006-01-02T15:04:05"
)

// loadActiveEntitiesAtDateHandler writes entities as json active for given moment, with matching attributes
func loadActiveEntitiesAtDateHandler(wrapper ServiceParameters, writer http.ResponseWriter, request *http.Request) error {
	defer request.Body.Close()

	trait := request.PathValue("trait")
	momentStr := request.PathValue("moment")
	var moment time.Time
	if t, err := time.Parse(DATE_WS_FORMAT, momentStr); err != nil {
		return NewServiceHttpClientError("invalid date: " + err.Error())
	} else {
		moment = t
	}

	values := request.URL.Query()
	queryValues := make(map[string]string)
	for k, v := range values {
		if len(v) != 1 {
			return NewServiceHttpClientError("invalid parameter: expecting key value, with one value per key")
		}

		queryValues[k] = v[0]
	}

	activeValues, errLoad := wrapper.Dao.LoadActiveEntitiesAtTime(wrapper.Ctx, moment, trait, queryValues)
	if errLoad != nil {
		return NewServiceInternalServerError("failed to load: " + errLoad.Error())
	}

	json.NewEncoder(writer).Encode(activeValues)
	return nil
}

// loadActiveEntitiesHandler writes entities as json active for now, with matching attributes
func loadActiveEntitiesHandler(wrapper ServiceParameters, writer http.ResponseWriter, request *http.Request) error {
	defer request.Body.Close()

	trait := request.PathValue("trait")

	values := request.URL.Query()
	queryValues := make(map[string]string)
	for k, v := range values {
		if len(v) != 1 {
			return NewServiceHttpClientError("invalid parameter: expecting key value, with one value per key")
		}

		queryValues[k] = v[0]
	}

	activeValues, errLoad := wrapper.Dao.LoadActiveEntitiesAtTime(wrapper.Ctx, time.Now().UTC(), trait, queryValues)
	if errLoad != nil {
		return NewServiceInternalServerError("failed to load: " + errLoad.Error())
	}

	json.NewEncoder(writer).Encode(activeValues)
	return nil
}
