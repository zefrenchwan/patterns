package serving

import "net/http"

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

// NewServiceHttpClientError returns a 400 error with a specific message
func NewServiceHttpClientError(message string) ServiceHttpError {
	return ServiceHttpError{
		httpCode: http.StatusBadRequest,
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

// NewServiceInternalServerError returns a 500 error with a specific message
func NewServiceInternalServerError(message string) ServiceHttpError {
	return ServiceHttpError{
		httpCode: http.StatusInternalServerError,
		message:  message,
	}
}
