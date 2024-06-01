package serving

import (
	"encoding/json"
	"io"
	"net/http"
)

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

	newId, errCreate := wrapper.Dao.CreateGraph(wrapper.Ctx, user, input.Name, input.Description, input.Sources)
	if errCreate != nil {
		return BuildApiErrorFromStorageError(errCreate)
	} else if len(input.Metadata) == 0 {
		json.NewEncoder(w).Encode(newId)
		return nil
	}

	errUpdate := wrapper.Dao.UpsertMetadataForGraph(wrapper.Ctx, user, newId, input.Metadata)
	if errUpdate != nil {
		return BuildApiErrorFromStorageError(errUpdate)
	}

	json.NewEncoder(w).Encode(newId)
	return nil
}
