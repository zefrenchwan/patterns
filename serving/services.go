package serving

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zefrenchwan/patterns.git/storage"
	"go.uber.org/zap"
)

const (
	// Expected date format
	URL_DATE_FORMAT = "2006-01-02T15:04:05"
)

// DeserializeTimeFromURL returns either a parsed time, or an error
func DeserializeTimeFromURL(value string) (time.Time, error) {
	var result time.Time
	if len(URL_DATE_FORMAT) != len(value) {
		return result, fmt.Errorf("invalid input, expecting %s", URL_DATE_FORMAT)
	}

	return time.Parse(URL_DATE_FORMAT, value)
}

// RequestContextKey is key type for context keys when using specific info (such as current user)
type RequestContextKey string

// InitService returns a new valid servemux to launch
func InitService(dao storage.Dao, initialContext context.Context, logger *zap.SugaredLogger) *http.ServeMux {
	mux := http.NewServeMux()

	parameters := ServiceParameters{
		Dao:    dao,
		Ctx:    initialContext,
		Logger: logger,
	}

	// TODO: add in here your own handlers
	// ADMIN PART
	AddGetServiceHandlerToMux(mux, "/status/", checkStatusHandler, parameters)
	AddPostServiceHandlerToMux(mux, "/token/", checkUserAndGenerateTokenHandler, parameters)
	AddAuthenticatedPostServiceHandlerToMux(mux, "/user/upsert/", upsertUserHandler, parameters)
	// GRAPHS OPERATIONS
	AddAuthenticatedPostServiceHandlerToMux(mux, "/graph/create/", createGraphHandler, parameters)
	AddAuthenticatedPutServiceHandlerToMux(mux, "/graph/import/{importGraph}/into/{baseGraph}/", addImportToExistingGraphHandler, parameters)
	AddAuthenticatedDeleteServiceHandlerToMux(mux, "/graph/delete/{graphId}/", deleteGraphHandler, parameters)
	AddAuthenticatedGetServiceHandlerToMux(mux, "/graph/list/", listGraphHandler, parameters)
	AddAuthenticatedGetServiceHandlerToMux(mux, "/graph/load/{graphId}/", loadGraphHandler, parameters)
	AddAuthenticatedGetServiceHandlerToMux(mux, "/graph/slice/{graphId}/since/{moment}/", loadGraphSinceHandler, parameters)
	AddAuthenticatedGetServiceHandlerToMux(mux, "/graph/slice/{graphId}/between/{start}/and/{end}/", loadGraphBetweenHandler, parameters)
	AddAuthenticatedGetServiceHandlerToMux(mux, "/graph/snapshot/{graphId}/at/{moment}/", snapshotGraphHandler, parameters)
	AddAuthenticatedDeleteServiceHandlerToMux(mux, "/graph/all/clear/", clearGraphsHandler, parameters)
	// ELEMENTS OPERATIONS
	AddAuthenticatedPutServiceHandlerToMux(mux, "/elements/copy/{elementId}/to/{graphId}/", createEquivalenceElementHandler, parameters)
	AddAuthenticatedGetServiceHandlerToMux(mux, "/elements/load/{elementId}/", loadElementByIdHandler, parameters)
	AddAuthenticatedPostServiceHandlerToMux(mux, "/elements/upsert/graph/{graphId}/", upsertElementInGraphHandler, parameters)
	AddAuthenticatedDeleteServiceHandlerToMux(mux, "/elements/delete/{elementId}/", deleteElementHandler, parameters)
	// LOCAL FIND OPERATIONS
	AddAuthenticatedGetServiceHandlerToMux(mux, "/find/neighbors/of/entities/for/trait/{trait}/", findElementFullPeriodHandler, parameters)
	AddAuthenticatedGetServiceHandlerToMux(mux, "/find/neighbors/of/entities/for/trait/{trait}/since/{start}/", findElementSinceHandler, parameters)
	AddAuthenticatedGetServiceHandlerToMux(mux, "/find/neighbors/of/entities/for/trait/{trait}/until/{end}/", findElementUntilHandler, parameters)
	AddAuthenticatedGetServiceHandlerToMux(mux, "/find/neighbors/of/entities/for/trait/{trait}/between/{start}/and/{end}/", findElementBetweenHandler, parameters)
	// END OF HANDLERS MODIFICATION
	// mux is complete, all handlers are set
	return mux
}

// AddGetServiceHandlerToMux adds an handler to to the current mux for a GET
func AddGetServiceHandlerToMux(mux *http.ServeMux, urlPattern string, handler ServiceHandler, parameters ServiceParameters) {
	AddServiceHandlerToMux(mux, "GET", urlPattern, false, handler, parameters)
}

// AddPostServiceHandlerToMux adds an handler to to the current mux for a POST
func AddPostServiceHandlerToMux(mux *http.ServeMux, urlPattern string, handler ServiceHandler, parameters ServiceParameters) {
	AddServiceHandlerToMux(mux, "POST", urlPattern, false, handler, parameters)
}

// AddAuthenticatedGetServiceHandlerToMux adds an handler to to the current mux for a GET
func AddAuthenticatedGetServiceHandlerToMux(mux *http.ServeMux, urlPattern string, handler ServiceHandler, parameters ServiceParameters) {
	AddServiceHandlerToMux(mux, "GET", urlPattern, true, handler, parameters)
}

// AddAuthenticatedPostServiceHandlerToMux adds an handler to to the current mux for a POST
func AddAuthenticatedPostServiceHandlerToMux(mux *http.ServeMux, urlPattern string, handler ServiceHandler, parameters ServiceParameters) {
	AddServiceHandlerToMux(mux, "POST", urlPattern, true, handler, parameters)
}

// AddAuthenticatedDeleteServiceHandlerToMux adds an handler to the current mux for a DELETE
func AddAuthenticatedDeleteServiceHandlerToMux(mux *http.ServeMux, urlPattern string, handler ServiceHandler, parameters ServiceParameters) {
	AddServiceHandlerToMux(mux, "DELETE", urlPattern, true, handler, parameters)
}

// AddAuthenticatedPutServiceHandlerToMux adds an handler to the current mux for a PUT
func AddAuthenticatedPutServiceHandlerToMux(mux *http.ServeMux, urlPattern string, handler ServiceHandler, parameters ServiceParameters) {
	AddServiceHandlerToMux(mux, "PUT", urlPattern, true, handler, parameters)
}

// AddServiceHandlerToMux adds an handler to current mux
func AddServiceHandlerToMux(mux *http.ServeMux, method string, urlPattern string, testAuth bool, handler ServiceHandler, parameters ServiceParameters) {
	handlerFunction := func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Method, method) {
			http.Error(w, "Expecting "+method, http.StatusBadRequest)
			return
		}

		// test if user is valid
		if testAuth {
			if login, auth, err := validateAuthentication(parameters, r); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			} else if !auth {
				http.Error(w, "should authenticate", http.StatusUnauthorized)
				return
			} else {
				parameters.Ctx = context.WithValue(parameters.Ctx, RequestContextKey("user"), login)
			}
		}

		// isolated call, to include time monitoring later
		errHandler := handler(parameters, w, r)
		if errHandler != nil {
			switch customError, ok := errHandler.(ServiceHttpError); ok {
			case true:
				http.Error(w, customError.Error(), customError.HttpCode())
			default:
				http.Error(w, "Internal error: "+errHandler.Error(), http.StatusInternalServerError)
			}
		}
	}

	// register url matching
	mux.HandleFunc(urlPattern, handlerFunction)
	// deal with /value/ <=> /value
	size := len(urlPattern)
	if strings.HasSuffix(urlPattern, "/") {
		mux.HandleFunc(urlPattern[0:size-1], handlerFunction)
	} else {
		mux.HandleFunc(urlPattern+"/", handlerFunction)
	}
}

// ServiceParameters contains all parameters to use for a service
type ServiceParameters struct {
	Dao    storage.Dao
	Ctx    context.Context
	Logger *zap.SugaredLogger
}

// ServiceHandler adds more parameters than usual handler function
type ServiceHandler func(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error

// CurrentUser returns the current user if any, and a boolean to explicit if found
func (sp ServiceParameters) CurrentUser() (string, bool) {
	switch userValue := sp.Ctx.Value(RequestContextKey("user")); userValue {
	case nil:
		return "", false
	default:
		return userValue.(string), true
	}
}
