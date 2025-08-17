package errors

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
)

var (
	// registry holds all registered errors
	registry   = make(map[string]*Error)
	registryMu sync.RWMutex

	// Standard errors exported for convenience
	ErrNotFound          = New(http.StatusNotFound, 1, "not found")
	ErrBadRequest        = New(http.StatusBadRequest, 1, "bad request")
	ErrUnauthorized      = New(http.StatusUnauthorized, 1, "unauthorized")
	ErrForbidden         = New(http.StatusForbidden, 1, "forbidden")
	ErrInternal          = New(http.StatusInternalServerError, 1, "internal server error")
	ErrConflict          = New(http.StatusConflict, 1, "conflict")
	ErrUnprocessable     = New(http.StatusUnprocessableEntity, 1, "unprocessable entity")
	ErrTooManyRequests   = New(http.StatusTooManyRequests, 1, "too many requests")
	ErrServiceUnavailable = New(http.StatusServiceUnavailable, 1, "service unavailable")
)

// Error represents a Kitsunium SDK error with HTTP and exit codes
type Error struct {
	httpCode int                    // HTTP status code
	exitCode int                    // Process exit code
	message  string                 // Error message
	tags     map[string]struct{}    // Tags for categorization
	cause    error                  // Underlying cause
	details  map[string]interface{} // Additional details
	mu       sync.RWMutex           // Protects tags and details
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

// HTTPCode returns the HTTP status code
func (e *Error) HTTPCode() int {
	return e.httpCode
}

// ExitCode returns the process exit code
func (e *Error) ExitCode() int {
	return e.exitCode
}

// Message returns the error message without cause
func (e *Error) Message() string {
	return e.message
}

// Cause returns the underlying error
func (e *Error) Cause() error {
	return e.cause
}

// Unwrap implements errors.Unwrap for error chain support
func (e *Error) Unwrap() error {
	return e.cause
}

// Is implements errors.Is for error comparison
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.httpCode == t.httpCode && 
	       e.exitCode == t.exitCode && 
	       e.message == t.message
}

// HasTag checks if the error contains a specific tag
func (e *Error) HasTag(tag string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.tags[tag]
	return exists
}

// AddTag adds one or more tags to the error (modifies in place)
func (e *Error) AddTag(tags ...string) *Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tags == nil {
		e.tags = make(map[string]struct{})
	}
	for _, tag := range tags {
		e.tags[tag] = struct{}{}
	}
	return e
}

// RemoveTag removes one or more tags from the error (modifies in place)
func (e *Error) RemoveTag(tags ...string) *Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, tag := range tags {
		delete(e.tags, tag)
	}
	return e
}

// WithTag returns a new error with additional tags
func (e *Error) WithTag(tags ...string) *Error {
	ne := e.clone()
	ne.mu.Lock()
	defer ne.mu.Unlock()
	if ne.tags == nil {
		ne.tags = make(map[string]struct{})
	}
	for _, tag := range tags {
		ne.tags[tag] = struct{}{}
	}
	return ne
}

// Tags returns a list of tags associated with the error
func (e *Error) Tags() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]string, 0, len(e.tags))
	for tag := range e.tags {
		result = append(result, tag)
	}
	return result
}

// WithCause wraps another error as the cause
func (e *Error) WithCause(cause error) *Error {
	ne := e.clone()
	ne.cause = cause
	return ne
}

// WithDetail adds a key-value detail to the error
func (e *Error) WithDetail(key string, value interface{}) *Error {
	ne := e.clone()
	ne.mu.Lock()
	defer ne.mu.Unlock()
	if ne.details == nil {
		ne.details = make(map[string]interface{})
	}
	ne.details[key] = value
	return ne
}

// WithDetails adds multiple key-value details to the error
func (e *Error) WithDetails(details map[string]interface{}) *Error {
	ne := e.clone()
	ne.mu.Lock()
	defer ne.mu.Unlock()
	if ne.details == nil {
		ne.details = make(map[string]interface{})
	}
	for k, v := range details {
		ne.details[k] = v
	}
	return ne
}

// Details returns all details associated with the error
func (e *Error) Details() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.details == nil {
		return nil
	}
	// Return a copy to prevent modification
	result := make(map[string]interface{}, len(e.details))
	for k, v := range e.details {
		result[k] = v
	}
	return result
}

// GetDetail returns a specific detail value
func (e *Error) GetDetail(key string) (interface{}, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.details == nil {
		return nil, false
	}
	val, ok := e.details[key]
	return val, ok
}

// clone creates a copy of the error
func (e *Error) clone() *Error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	ne := &Error{
		httpCode: e.httpCode,
		exitCode: e.exitCode,
		message:  e.message,
		cause:    e.cause,
	}
	
	if e.tags != nil {
		ne.tags = make(map[string]struct{}, len(e.tags))
		for tag := range e.tags {
			ne.tags[tag] = struct{}{}
		}
	}
	
	if e.details != nil {
		ne.details = make(map[string]interface{}, len(e.details))
		for k, v := range e.details {
			ne.details[k] = v
		}
	}
	
	return ne
}

// New creates a new error with HTTP code, exit code, and message
func New(httpCode, exitCode int, message string, tags ...string) *Error {
	tagSet := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tagSet[tag] = struct{}{}
	}

	err := &Error{
		httpCode: httpCode,
		exitCode: exitCode,
		message:  message,
		tags:     tagSet,
	}

	// Register the error
	registryMu.Lock()
	registry[message] = err
	registryMu.Unlock()

	return err
}

// Newf creates a new error with formatted message
func Newf(httpCode, exitCode int, format string, args ...interface{}) *Error {
	return New(httpCode, exitCode, fmt.Sprintf(format, args...))
}

// Wrap wraps an existing error with SDK error information
func Wrap(err error, httpCode, exitCode int, message string) *Error {
	if err == nil {
		return nil
	}
	
	// If it's already an SDK error, preserve its information
	if sdkErr, ok := err.(*Error); ok {
		return sdkErr.WithCause(errors.New(message))
	}
	
	return &Error{
		httpCode: httpCode,
		exitCode: exitCode,
		message:  message,
		cause:    err,
		tags:     make(map[string]struct{}),
	}
}

// Wrapf wraps an existing error with formatted message
func Wrapf(err error, httpCode, exitCode int, format string, args ...interface{}) *Error {
	return Wrap(err, httpCode, exitCode, fmt.Sprintf(format, args...))
}

// Is checks if an error is of a specific type
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Unwrap returns the result of calling the Unwrap method on err
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// ListErrors returns all registered errors
func ListErrors() map[string]*Error {
	registryMu.RLock()
	defer registryMu.RUnlock()
	
	result := make(map[string]*Error, len(registry))
	for k, v := range registry {
		result[k] = v
	}
	return result
}

// GetError retrieves a registered error by message
func GetError(message string) (*Error, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	err, ok := registry[message]
	return err, ok
}

// ClearRegistry clears all registered errors (useful for testing)
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]*Error)
}

// HTTPStatusText returns the standard text for an HTTP status code
func HTTPStatusText(code int) string {
	return http.StatusText(code)
}

// IsClientError returns true if the HTTP code is a client error (4xx)
func (e *Error) IsClientError() bool {
	return e.httpCode >= 400 && e.httpCode < 500
}

// IsServerError returns true if the HTTP code is a server error (5xx)
func (e *Error) IsServerError() bool {
	return e.httpCode >= 500 && e.httpCode < 600
}

// Code returns the HTTP code (for backward compatibility)
func (e *Error) Code() uint16 {
	return uint16(e.httpCode)
}