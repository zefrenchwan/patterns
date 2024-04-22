package serving

import (
	"encoding/json"
	"net/http"

	"github.com/zefrenchwan/patterns.git/storage"
)

// loadElementHandler returns, if any, the element matching its id as a json
func loadElementHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	id := r.PathValue("id")

	element, errElement := wrapper.Dao.LoadElementById(wrapper.Ctx, id)
	if errElement != nil {
		return NewServiceInternalServerError(errElement.Error())
	} else if element == nil {
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte{})
	} else {
		dto := storage.SerializeElement(element)
		json.NewEncoder(w).Encode(dto)
	}

	return nil
}
