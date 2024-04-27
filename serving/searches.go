package serving

import (
	"encoding/json"
	"net/http"
)

// searchSynonymsHandler returns all entities id, period and traits matching query given in parameters
func searchSynonymsHandler(wrapper ServiceParameters, writer http.ResponseWriter, request *http.Request) error {
	defer request.Body.Close()

	values := request.URL.Query()
	queryValues := make(map[string]string)
	for k, v := range values {
		if len(v) != 1 {
			return NewServiceHttpClientError("invalid parameter: expecting key value, with one value per key")
		}

		queryValues[k] = v[0]
	}

	dtos, errorLoad := wrapper.Dao.LoadEntitiesTraits(wrapper.Ctx, queryValues)
	if errorLoad != nil {
		return NewServiceInternalServerError(errorLoad.Error())
	}

	json.NewEncoder(writer).Encode(dtos)
	return nil
}
