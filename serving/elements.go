package serving

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/zefrenchwan/patterns.git/storage"
)

// upsertElementInGraphHandler loads an element dto and then saves it to database
func upsertElementInGraphHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	graphId := r.PathValue("graphId")
	if len(graphId) == 0 {
		return NewServiceHttpClientError("expecting graph id")
	}

	var input storage.ElementDTO
	if body, err := io.ReadAll(r.Body); err != nil {
		return NewServiceInternalServerError(err.Error())
	} else if errM := json.Unmarshal(body, &input); errM != nil {
		return NewServiceInternalServerError(errM.Error())
	} else if len(input.Id) == 0 {
		return NewServiceHttpClientError("expecting element id")
	}

	element, errElement := storage.DeserializeElement(input)
	if errElement != nil {
		message := fmt.Sprintf("invalid json: %s", errElement.Error())
		return NewServiceHttpClientError(message)
	}

	if err := wrapper.Dao.UpsertElement(wrapper.Ctx, user, graphId, element); err != nil {
		return NewServiceInternalServerError(err.Error())
	}

	w.WriteHeader(200)
	return nil
}
