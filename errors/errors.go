package errors

import (
	"fmt"
	"strings"
	"time"
)

// SDKError is the base error type for all SDK errors.
type SDKError struct {
	Message   string      `json:"message"`
	Code      string      `json:"code"`
	Details   interface{} `json:"details"`
	Timestamp string      `json:"timestamp"`
}

func (e *SDKError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewSDKError creates a new SDKError.
func NewSDKError(message string, code string, details interface{}) *SDKError {
	return &SDKError{
		Message:   message,
		Code:      code,
		Details:   SanitizeDetails(details),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// SanitizeDetails removes sensitive fields from error details.
func SanitizeDetails(details interface{}) interface{} {
	if details == nil {
		return nil
	}

	switch val := details.(type) {
	case map[string]interface{}:
		sanitized := make(map[string]interface{})
		for k, v := range val {
			lowerK := strings.ToLower(k)
			if isSensitiveKey(lowerK) {
				sanitized[k] = "[REDACTED]"
			} else {
				sanitized[k] = SanitizeDetails(v)
			}
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, len(val))
		for i, item := range val {
			sanitized[i] = SanitizeDetails(item)
		}
		return sanitized
	default:
		return val
	}
}

func isSensitiveKey(key string) bool {
	sensitiveKeys := []string{"auth_token", "token", "password", "secret", "api_key", "apikey", "authorization"}
	for _, sk := range sensitiveKeys {
		if strings.Contains(key, sk) {
			return true
		}
	}
	return false
}

// AuthenticationError represents 401 or 403 HTTP errors.
type AuthenticationError struct {
	*SDKError
}

func NewAuthenticationError(message string, details interface{}) *AuthenticationError {
	if message == "" {
		message = "Falha na autenticação"
	}
	return &AuthenticationError{
		SDKError: NewSDKError(message, "AUTHENTICATION_ERROR", details),
	}
}

func (e *AuthenticationError) Error() string {
	return e.SDKError.Error()
}

func (e *AuthenticationError) As(target interface{}) bool {
	if sdkErr, ok := target.(**SDKError); ok {
		*sdkErr = e.SDKError
		return true
	}
	return false
}

// ValidationError represents 400 or 422 HTTP errors.
type ValidationError struct {
	*SDKError
}

func NewValidationError(message string, details interface{}) *ValidationError {
	if message == "" {
		message = "Erro de validação"
	}
	return &ValidationError{
		SDKError: NewSDKError(message, "VALIDATION_ERROR", details),
	}
}

func (e *ValidationError) Error() string {
	return e.SDKError.Error()
}

func (e *ValidationError) As(target interface{}) bool {
	if sdkErr, ok := target.(**SDKError); ok {
		*sdkErr = e.SDKError
		return true
	}
	return false
}

// RateLimitError represents 429 HTTP errors.
type RateLimitError struct {
	*SDKError
	RetryAfter int `json:"retryAfter"`
}

func NewRateLimitError(message string, retryAfter int, details interface{}) *RateLimitError {
	if message == "" {
		message = "Limite de requisições excedido"
	}
	return &RateLimitError{
		SDKError:   NewSDKError(message, "RATE_LIMIT_ERROR", details),
		RetryAfter: retryAfter,
	}
}

func (e *RateLimitError) Error() string {
	return e.SDKError.Error()
}

func (e *RateLimitError) As(target interface{}) bool {
	if sdkErr, ok := target.(**SDKError); ok {
		*sdkErr = e.SDKError
		return true
	}
	return false
}

// EndpointNotFoundError represents when the requested endpoint slug does not exist.
type EndpointNotFoundError struct {
	*SDKError
}

func NewEndpointNotFoundError(message string, details interface{}) *EndpointNotFoundError {
	if message == "" {
		message = "Endpoint não encontrado"
	}
	return &EndpointNotFoundError{
		SDKError: NewSDKError(message, "ENDPOINT_NOT_FOUND", details),
	}
}

func (e *EndpointNotFoundError) Error() string {
	return e.SDKError.Error()
}

func (e *EndpointNotFoundError) As(target interface{}) bool {
	if sdkErr, ok := target.(**SDKError); ok {
		*sdkErr = e.SDKError
		return true
	}
	return false
}

// SpecificationError represents an error reading/validating the Postman spec or manifest.
type SpecificationError struct {
	*SDKError
}

func NewSpecificationError(message string, details interface{}) *SpecificationError {
	if message == "" {
		message = "Erro na especificação da API"
	}
	return &SpecificationError{
		SDKError: NewSDKError(message, "SPECIFICATION_ERROR", details),
	}
}

func (e *SpecificationError) Error() string {
	return e.SDKError.Error()
}

func (e *SpecificationError) As(target interface{}) bool {
	if sdkErr, ok := target.(**SDKError); ok {
		*sdkErr = e.SDKError
		return true
	}
	return false
}
