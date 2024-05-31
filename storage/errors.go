package storage

import "strings"

const (
	AUTH_PREFIX     = "auth failure:"
	RESOURCE_PREFIX = "resource not found: "
)

// IsAuthErrorMessage returns true if message is about an auth issue
func IsAuthErrorMessage(message string) bool {
	return strings.HasPrefix(message, AUTH_PREFIX)
}

// IsResourceNotFoundMessage returns true if message is about a resource that was not found
func IsResourceNotFoundMessage(message string) bool {
	return strings.HasPrefix(RESOURCE_PREFIX, message)
}
