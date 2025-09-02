package services

import "errors"

// Common service errors
var (
	// ErrNotFound indicates a resource was not found
	ErrNotFound = errors.New("resource not found")
	
	// ErrAlreadyExists indicates a resource already exists
	ErrAlreadyExists = errors.New("resource already exists")
	
	// ErrInvalidInput indicates invalid input data
	ErrInvalidInput = errors.New("invalid input")
	
	// ErrUnauthorized indicates unauthorized access
	ErrUnauthorized = errors.New("unauthorized")
	
	// ErrForbidden indicates forbidden access
	ErrForbidden = errors.New("forbidden")
	
	// ErrConflict indicates a conflict with current state
	ErrConflict = errors.New("conflict")
	
	// ErrHasAssociatedResources indicates the resource has associated resources that prevent deletion
	ErrHasAssociatedResources = errors.New("resource has associated resources")
	
	// ErrValidationFailed indicates input validation failed
	ErrValidationFailed = errors.New("validation failed")
	
	// ErrInternalError indicates an internal server error
	ErrInternalError = errors.New("internal server error")
)

// IsNotFound checks if error is ErrNotFound
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsAlreadyExists checks if error is ErrAlreadyExists
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// IsInvalidInput checks if error is ErrInvalidInput
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsUnauthorized checks if error is ErrUnauthorized
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden checks if error is ErrForbidden
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsConflict checks if error is ErrConflict
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsValidationFailed checks if error is ErrValidationFailed
func IsValidationFailed(err error) bool {
	return errors.Is(err, ErrValidationFailed)
}