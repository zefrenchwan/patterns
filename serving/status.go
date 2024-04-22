package serving

import (
	"encoding/json"
	"net/http"
)

// CheckStatusResponse defines json to display when asking for status
type CheckStatusResponse struct {
	// Active is true when server is up
	Active bool `json:"active"`
	// Description is more about this serving instance
	Description string `json:"description,omitempty"`
}

// checkStatusHandler deals with a request to test status on a server
func checkStatusHandler(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	result := CheckStatusResponse{
		Active:      true,
		Description: "Patterns server",
	}

	json.NewEncoder(w).Encode(result)
	return nil
}
