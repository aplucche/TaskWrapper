package main

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypePermission   ErrorType = "permission"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeConflict     ErrorType = "conflict"
	ErrorTypeInternal     ErrorType = "internal"
	ErrorTypeExternal     ErrorType = "external"
	ErrorTypeTimeout      ErrorType = "timeout"
	ErrorTypeCancelled    ErrorType = "cancelled"
	ErrorTypeUnsupported  ErrorType = "unsupported"
)

// AppError provides structured error information
type AppError struct {
	Type       ErrorType
	Message    string
	Err        error
	Context    map[string]interface{}
	StackTrace string
	Retryable  bool
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error
func NewAppError(errType ErrorType, message string, err error) *AppError {
	return &AppError{
		Type:       errType,
		Message:    message,
		Err:        err,
		Context:    make(map[string]interface{}),
		StackTrace: getStackTrace(2),
		Retryable:  isRetryable(errType),
	}
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	e.Context[key] = value
	return e
}

// WithRetryable sets whether the error is retryable
func (e *AppError) WithRetryable(retryable bool) *AppError {
	e.Retryable = retryable
	return e
}

// getStackTrace captures the current stack trace
func getStackTrace(skip int) string {
	var sb strings.Builder
	for i := skip; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			sb.WriteString(fmt.Sprintf("%s:%d %s\n", file, line, fn.Name()))
		}
	}
	return sb.String()
}

// isRetryable determines if an error type is typically retryable
func isRetryable(errType ErrorType) bool {
	switch errType {
	case ErrorTypeTimeout, ErrorTypeExternal:
		return true
	default:
		return false
	}
}

// Common error constructors

// ValidationError creates a validation error
func ValidationError(message string, err error) *AppError {
	return NewAppError(ErrorTypeValidation, message, err)
}

// PermissionError creates a permission error
func PermissionError(message string, err error) *AppError {
	return NewAppError(ErrorTypePermission, message, err)
}

// NotFoundError creates a not found error
func NotFoundError(message string, err error) *AppError {
	return NewAppError(ErrorTypeNotFound, message, err)
}

// ConflictError creates a conflict error
func ConflictError(message string, err error) *AppError {
	return NewAppError(ErrorTypeConflict, message, err)
}

// InternalError creates an internal error
func InternalError(message string, err error) *AppError {
	return NewAppError(ErrorTypeInternal, message, err)
}

// TimeoutError creates a timeout error
func TimeoutError(message string, err error) *AppError {
	return NewAppError(ErrorTypeTimeout, message, err)
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	logger Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// Handle processes an error appropriately
func (eh *ErrorHandler) Handle(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's an AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		eh.logAppError(appErr)
		return appErr
	}

	// Create AppError from regular error
	appErr = InternalError("unexpected error", err)
	eh.logAppError(appErr)
	return appErr
}

// HandleWithRecovery handles an error and attempts recovery
func (eh *ErrorHandler) HandleWithRecovery(err error, recoveryFn func() error) error {
	if err == nil {
		return nil
	}

	handledErr := eh.Handle(err)
	
	// Check if error is retryable
	var appErr *AppError
	if errors.As(handledErr, &appErr) && appErr.Retryable && recoveryFn != nil {
		eh.logger.Info("Attempting recovery for retryable error")
		if recoveryErr := recoveryFn(); recoveryErr != nil {
			eh.logger.Error("Recovery failed", recoveryErr)
			return appErr
		}
		eh.logger.Info("Recovery successful")
		return nil
	}

	return handledErr
}

// logAppError logs an AppError with all context
func (eh *ErrorHandler) logAppError(err *AppError) {
	fields := map[string]interface{}{
		"error_type": err.Type,
		"retryable":  err.Retryable,
	}
	
	// Add error context
	for k, v := range err.Context {
		fields[k] = v
	}
	
	// Add stack trace for internal errors
	if err.Type == ErrorTypeInternal {
		fields["stack_trace"] = err.StackTrace
	}
	
	eh.logger.ErrorWithFields(err.Message, err.Err, fields)
}

// RecoverPanic recovers from panics and converts them to errors
func (eh *ErrorHandler) RecoverPanic() {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic recovered: %v", r)
		appErr := InternalError("panic occurred", err).
			WithContext("panic_value", r).
			WithContext("stack_trace", getStackTrace(3))
		eh.logAppError(appErr)
	}
}

// WithRecover wraps a function with panic recovery
func (eh *ErrorHandler) WithRecover(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = InternalError("panic in function", fmt.Errorf("%v", r))
			eh.Handle(err)
		}
	}()
	return fn()
}