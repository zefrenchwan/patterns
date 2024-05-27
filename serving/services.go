package serving

import (
	"context"
	"net/http"
	"strings"

	"github.com/zefrenchwan/patterns.git/storage"
	"go.uber.org/zap"
)

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

// AddServiceHandlerToMux adds an handler to current mux
func AddServiceHandlerToMux(mux *http.ServeMux, method string, urlPattern string, testAuth bool, handler ServiceHandler, parameters ServiceParameters) {
	mux.HandleFunc(urlPattern, func(w http.ResponseWriter, r *http.Request) {
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

		if err := handler(parameters, w, r); err != nil {
			switch customError, ok := err.(ServiceHttpError); ok {
			case true:
				http.Error(w, customError.Error(), customError.HttpCode())
			default:
				http.Error(w, "Internal error: "+err.Error(), http.StatusInternalServerError)
			}
		}
	})
}

// ServiceParameters contains all parameters to use for a service
type ServiceParameters struct {
	Dao    storage.Dao
	Ctx    context.Context
	Logger *zap.SugaredLogger
}

// ServiceHandler adds more parameters than usual handler function
type ServiceHandler func(wrapper ServiceParameters, w http.ResponseWriter, r *http.Request) error
