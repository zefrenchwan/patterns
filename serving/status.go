package serving

import (
	"encoding/json"
	"net/http"
)

type CheckStatusResponse struct {
	Active      bool   `json:"active"`
	Description string `json:"description,omitempty"`
}

func CheckStatusHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	result := CheckStatusResponse{
		Active:      true,
		Description: "Patterns server",
	}

	json.NewEncoder(w).Encode(result)
	return nil
}
