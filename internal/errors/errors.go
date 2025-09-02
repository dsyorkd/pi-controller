package errors

import (
	"errors"
	"fmt"
)

// Common application error types
var (
	// ErrNotFound indicates a resource was not found
	ErrNotFound = errors.New("resource not found")
	
	// ErrAlreadyExists indicates a resource already exists
	ErrAlreadyExists = errors.New("resource already exists")
	
	// ErrInvalidInput indicates invalid input was provided
	ErrInvalidInput = errors.New("invalid input")
	
	// ErrUnauthorized indicates insufficient permissions
	ErrUnauthorized = errors.New("unauthorized")
	
	// ErrForbidden indicates access is forbidden
	ErrForbidden = errors.New("forbidden")
	
	// ErrConflict indicates a resource conflict
	ErrConflict = errors.New("resource conflict")
	
	// ErrInternal indicates an internal server error
	ErrInternal = errors.New("internal server error")
	
	// ErrServiceUnavailable indicates a service is unavailable
	ErrServiceUnavailable = errors.New("service unavailable")
)

// DatabaseError represents database-specific errors
type DatabaseError struct {
	Operation string
	Err       error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database %s failed: %v", e.Operation, e.Err)
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// NewDatabaseError creates a new database error
func NewDatabaseError(operation string, err error) *DatabaseError {
	return &DatabaseError{
		Operation: operation,
		Err:       err,
	}
}

// ValidationError represents validation-specific errors
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s' with value '%v': %s", e.Field, e.Value, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// APIError represents API-specific errors with HTTP status codes
type APIError struct {
	Code    int
	Message string
	Err     error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("API error %d: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// NewAPIError creates a new API error
func NewAPIError(code int, message string, err error) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// GPIOError represents GPIO-specific errors
type GPIOError struct {
	Pin       int
	Operation string
	Err       error
}

func (e *GPIOError) Error() string {
	return fmt.Sprintf("GPIO pin %d %s failed: %v", e.Pin, e.Operation, e.Err)
}

func (e *GPIOError) Unwrap() error {
	return e.Err
}

// NewGPIOError creates a new GPIO error
func NewGPIOError(pin int, operation string, err error) *GPIOError {
	return &GPIOError{
		Pin:       pin,
		Operation: operation,
		Err:       err,
	}
}

// NetworkError represents network-specific errors
type NetworkError struct {
	Host      string
	Operation string
	Err       error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network %s to %s failed: %v", e.Operation, e.Host, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// NewNetworkError creates a new network error
func NewNetworkError(host, operation string, err error) *NetworkError {
	return &NetworkError{
		Host:      host,
		Operation: operation,
		Err:       err,
	}
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with formatted additional context
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// Is checks if an error matches a target error
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As checks if an error can be assigned to target
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}