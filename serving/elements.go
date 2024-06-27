package serving

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/zefrenchwan/patterns.git/storage"
)

// loadElementByIdHandler loads an element by id and returns matching JSON
func loadElementByIdHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	elementId := r.PathValue("elementId")
	if len(elementId) == 0 {
		return NewServiceHttpClientError("expecting element id")
	}

	element, errLoad := wrapper.Dao.LoadElementForUser(wrapper.Ctx, user, elementId)
	if errLoad != nil {
		return BuildApiErrorFromStorageError(errLoad)
	} else if element == nil {
		w.WriteHeader(404)
		return nil
	} else if response, err := storage.SerializeElement(element); err != nil {
		return NewServiceInternalServerError(err.Error())
	} else if err := json.NewEncoder(w).Encode(response); err != nil {
		return NewServiceInternalServerError(err.Error())
	}

	return nil
}

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

// deleteElementHandler deletes an element in the graph
func deleteElementHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	elementId := r.PathValue("elementId")
	if len(elementId) == 0 {
		return NewServiceHttpClientError("expecting element id")
	}

	if err := wrapper.Dao.DeleteElement(wrapper.Ctx, user, elementId); err != nil {
		return NewServiceInternalServerError(err.Error())
	}

	w.WriteHeader(200)
	return nil
}

// createEquivalenceElementHandler copies an element to another graph and returns its id
func createEquivalenceElementHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	user, auth := wrapper.CurrentUser()
	if !auth {
		return NewServiceForbiddenError("should authenticate")
	}

	elementId := r.PathValue("elementId")
	if len(elementId) == 0 {
		return NewServiceHttpClientError("expecting element id")
	}

	graphId := r.PathValue("graphId")
	if len(graphId) == 0 {
		return NewServiceHttpClientError("expecting graph id")
	}

	newElementId := uuid.NewString()

	if err := wrapper.Dao.CreateEquivalentElement(wrapper.Ctx, user, elementId, graphId, newElementId); err != nil {
		return NewServiceInternalServerError(err.Error())
	} else if err := json.NewEncoder(w).Encode(newElementId); err != nil {
		return NewServiceInternalServerError(err.Error())
	}

	return nil
}
