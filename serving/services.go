package serving

import (
	"context"
	"net/http"
	"strings"

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
	// ADMIN PART
	AddGetServiceHandlerToMux(mux, "/status/", checkStatusHandler, parameters)
	// UPSERT PART
	AddPostServiceHandlerToMux(mux, "/upsert/elements/", upsertElementsHandler, parameters)
	// LOAD PART
	AddGetServiceHandlerToMux(mux, "/load/elements/id/{id}", loadElementHandler, parameters)
	// SNAPSHOT PART
	AddGetServiceHandlerToMux(mux, "/snapshot/entities/trait/{trait}", loadActiveEntitiesHandler, parameters)
	AddGetServiceHandlerToMux(mux, "/snapshot/entities/trait/{trait}/moment/{moment}", loadActiveEntitiesAtDateHandler, parameters)
	AddGetServiceHandlerToMux(mux, "/snapshot/neighbors/counters/id/{id}", loadRelationsStatsAroundElementHandler, parameters)
	AddGetServiceHandlerToMux(mux, "/snapshot/neighbors/counters/id/{id}/moment/{moment}", loadRelationsStatsAroundElementAtDateHandler, parameters)
	AddGetServiceHandlerToMux(mux, "/snapshot/neighbors/counters/operands/id/{id}", loadRelationsStatsWithOperandsAroundElementHandler, parameters)
	AddGetServiceHandlerToMux(mux, "/snapshot/neighbors/counters/operands/id/{id}/moment/{moment}", loadRelationsStatsWithOperandsAroundElementAtDateHandler, parameters)
	// END OF HANDLERS MODIFICATION

	// mux is complete, all handlers are set
	return mux
}

// AddGetServiceHandlerToMux adds an handler to to the current mux for a GET
func AddGetServiceHandlerToMux(mux *http.ServeMux, urlPattern string, handler ServiceHandler, parameters ServiceParameters) {
	AddServiceHandlerToMux(mux, "GET", urlPattern, handler, parameters)
}

// AddPostServiceHandlerToMux adds an handler to to the current mux for a POST
func AddPostServiceHandlerToMux(mux *http.ServeMux, urlPattern string, handler ServiceHandler, parameters ServiceParameters) {
	AddServiceHandlerToMux(mux, "POST", urlPattern, handler, parameters)
}

// AddServiceHandlerToMux adds an handler to current mux
func AddServiceHandlerToMux(mux *http.ServeMux, method string, urlPattern string, handler ServiceHandler, parameters ServiceParameters) {
	mux.HandleFunc(urlPattern, func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Method, method) {
			http.Error(w, "Expecting "+method, 400)
		} else if err := handler(parameters, w, r); err != nil {
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
