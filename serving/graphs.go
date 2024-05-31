package serving

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/zefrenchwan/patterns.git/storage"
)

type GraphCreationInput struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Metadata    map[string][]string `json:"metadata,omitempty"`
	Sources     []string            `json:"sources,omitempty"`
}

func createGraphHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	var input GraphCreationInput
	if body, err := io.ReadAll(r.Body); err != nil {
		return NewServiceInternalServerError(err.Error())
	} else if errM := json.Unmarshal(body, &input); errM != nil {
		return NewServiceInternalServerError(errM.Error())
	} else if len(input.Name) == 0 {
		return NewServiceHttpClientError("expecting graph name")
	}

	newId, errCreate := wrapper.Dao.CreateGraph(wrapper.Ctx, user, input.Name, input.Description, input.Sources)
	if errCreate == nil {
		json.NewEncoder(w).Encode(newId)
		return nil
	}

	message := errCreate.Error()
	switch {
	case storage.IsAuthErrorMessage(message):
		return NewServiceForbiddenError(message)
	case storage.IsResourceNotFoundMessage(message):
		return NewServiceNotFoundError(message)
	default:
		return NewServiceInternalServerError(message)
	}
}
