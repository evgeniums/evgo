package auth

import (
	"net/http"
)

const (
	ErrorCodeUnauthorized          string = "unauthorized"
	ErrorCodeInvalidAuthSchema     string = "invalid_auth_schema"
	ErrorCodeUnsupportedAuthMethod string = "unknown_auth_method"
)

var ErrorDescriptions = map[string]string{
	ErrorCodeUnauthorized:          "Request is not authorized",
	ErrorCodeInvalidAuthSchema:     "Invalid authorization schema",
	ErrorCodeUnsupportedAuthMethod: "Unsupported authorization method",
}

var ErrorHttpCodes = map[string]int{
	ErrorCodeUnauthorized:          http.StatusUnauthorized,
	ErrorCodeInvalidAuthSchema:     http.StatusInternalServerError, // because this is error of server configuration
	ErrorCodeUnsupportedAuthMethod: http.StatusUnauthorized,
}
