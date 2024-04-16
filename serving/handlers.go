package serving

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/zefrenchwan/patterns.git/storage"
)

func InitService(mux *http.ServeMux, dao storage.Dao) {
	mux.HandleFunc("/load/trait/{trait}", func(w http.ResponseWriter, r *http.Request) {
		if err := LoadActiveEntitiesHandler(dao, w, r); err != nil {
			http.Error(w, "Internal error: "+err.Error(), 500)
		}
	})
}

func LoadActiveEntitiesHandler(dao storage.Dao, writer http.ResponseWriter, request *http.Request) error {
	defer request.Body.Close()

	trait := request.PathValue("trait")

	values := request.URL.Query()
	queryValues := make(map[string]string)
	for k, v := range values {
		queryValues[k] = v[0]
	}

	activeValues, errLoad := dao.LoadActiveEntitiesAtTime(context.Background(), time.Now().UTC(), trait, queryValues)
	if errLoad != nil {
		return errLoad
	}

	json.NewEncoder(writer).Encode(activeValues)
	return nil

}
