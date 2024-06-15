package serving

import (
	"net/http"
	"strings"

	"github.com/zefrenchwan/patterns.git/storage"
)

// ServiceHttpError is a custom error with an http code to return
type ServiceHttpError struct {
	httpCode int
	message  string
}

// Error to implement error interface
func (e ServiceHttpError) Error() string {
	return e.message
}

// HttpCode returns http code for response
func (e ServiceHttpError) HttpCode() int {
	return e.httpCode
}

func BuildApiErrorFromStorageError(sourceError error) error {
	if sourceError == nil {
		return sourceError
	}

	message := strings.Trim(sourceError.Error(), " ")

	switch storage.FindCodeInPSQLException(sourceError) {
	case storage.INCONSISTENCY_CODE:
		return NewServiceForbiddenError(message)
	case storage.AUTH_CODE:
		return NewServiceUnauthorizedError(message)
	case storage.RESOURCE_CODE:
		return NewServiceNotFoundError(message)
	default:
		return NewServiceInternalServerError(message)
	}
}

// NewServiceHttpClientError returns a 400 error with a specific message
func NewServiceHttpClientError(message string) ServiceHttpError {
	return ServiceHttpError{
		httpCode: http.StatusBadRequest,
		message:  message,
	}
}

// NewServiceUnauthorizedError returns a new 401 (unauthorized) error
func NewServiceUnauthorizedError(message string) ServiceHttpError {
	return ServiceHttpError{
		httpCode: http.StatusUnauthorized,
		message:  message,
	}
}

// NewServiceForbiddenError returns a new 403 (forbidden) error
func NewServiceForbiddenError(message string) ServiceHttpError {
	return ServiceHttpError{
		httpCode: http.StatusForbidden,
		message:  message,
	}
}

// NewServiceUnprocessableEntityError returns a 422 error (unprocessable)
func NewServiceUnprocessableEntityError(message string) ServiceHttpError {
	return ServiceHttpError{
		httpCode: http.StatusUnprocessableEntity,
		message:  message,
	}
}

// NewServiceNotFoundError returns a 404 error with a specific message
func NewServiceNotFoundError(message string) ServiceHttpError {
	return ServiceHttpError{
		httpCode: http.StatusNotFound,
		message:  message,
	}
}

// NewServiceInternalServerError returns a 500 error with a specific message
func NewServiceInternalServerError(message string) ServiceHttpError {
	return ServiceHttpError{
		httpCode: http.StatusInternalServerError,
		message:  message,
	}
}
