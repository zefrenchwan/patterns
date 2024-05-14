package serving

import (
	"encoding/json"
	"io"
	"net/http"
)

type CreateGraphInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// createGraphHandler creates a graph with given name and description.
// It returns the id of the new graph
func createGraphHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	var currentUser string
	if currentValue := wrapper.Ctx.Value(RequestContextKey("user")); currentValue == nil {
		return NewServiceInternalServerError("expecting user")
	} else {
		currentUser = currentValue.(string)
	}

	var graphInput CreateGraphInput
	if body, errBody := io.ReadAll(r.Body); errBody != nil {
		return NewServiceUnprocessableEntityError(errBody.Error())
	} else if err := json.Unmarshal(body, &graphInput); err != nil {
		return NewServiceUnprocessableEntityError(err.Error())
	}

	var graphId string
	if id, err := wrapper.Dao.CreateGraph(wrapper.Ctx, currentUser, graphInput.Name, graphInput.Description); err != nil {
		return NewServiceInternalServerError(err.Error())
	} else {
		graphId = id
	}

	json.NewEncoder(w).Encode(graphId)
	return nil
}
