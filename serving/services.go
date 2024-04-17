package serving

import (
	"context"
	"net/http"

	"github.com/zefrenchwan/patterns.git/storage"
)

// InitService returns a new valid servemux to launch
func InitService(dao storage.Dao) *http.ServeMux {
	mux := http.NewServeMux()

	parameters := ServiceParameters{
		Dao: dao,
		Ctx: context.Background(),
	}

	// TODO: add in here your own handlers
	AddServiceHandlerToMux(mux, "/snapshot/trait/{trait}", loadActiveEntitiesHandler, parameters)
	AddServiceHandlerToMux(mux, "/snapshot/trait/{trait}/moment/{moment}", loadActiveEntitiesAtDateHandler, parameters)

	// mux is complete, all handlers are set
	return mux
}

// AddServiceHandlerToMux adds an handler to current mux
func AddServiceHandlerToMux(mux *http.ServeMux, urlPattern string, handler ServiceHandler, parameters ServiceParameters) {
	mux.HandleFunc(urlPattern, func(w http.ResponseWriter, r *http.Request) {
		if err := handler(parameters, w, r); err != nil {
			switch customError, ok := err.(ServiceHttpError); ok {
			case true:
				http.Error(w, customError.Error(), customError.HttpCode())
			default:
				http.Error(w, "Internal error: "+err.Error(), 500)
			}
		}
	})
}

// ServiceParameters contains all parameters to use for a service
type ServiceParameters struct {
	Dao storage.Dao
	Ctx context.Context
}

// ServiceHandler adds more parameters than usual handler function
type ServiceHandler func(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error
