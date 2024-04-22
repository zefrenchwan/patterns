package serving

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/zefrenchwan/patterns.git/storage"
)

// upsertElementsHandler receives a POST containing a list of dto and inserts them all
func upsertElementsHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	payload, errPayload := io.ReadAll(r.Body)
	if errPayload != nil {
		return NewServiceUnprocessableEntityError(errPayload.Error())
	}

	var elements []storage.ElementDTO
	if err := json.Unmarshal(payload, &elements); err != nil {
		return NewServiceUnprocessableEntityError(err.Error())
	}

	var globalErr error
	for _, dto := range elements {
		if element, err := storage.DeserializeElement(dto); err != nil {
			globalErr = errors.Join(globalErr, err)
			continue
		} else if err := wrapper.Dao.UpsertElement(wrapper.Ctx, element); err != nil {
			globalErr = errors.Join(globalErr, err)
			break
		}
	}

	if globalErr == nil {
		return globalErr
	} else {
		return NewServiceInternalServerError(globalErr.Error())
	}
}
