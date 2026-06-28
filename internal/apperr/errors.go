package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	Status  int
	Code    string
	Message string
	Details any
	Err     error
}

func New(status int, code string, message string) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

func NewWithDetails(status int, code string, message string, details any) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
	}
}

func Wrap(err error, status int, code string, message string) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func BadRequest(code string, message string) *Error {
	return New(http.StatusBadRequest, code, message)
}

func BadRequestWithDetails(code string, message string, details any) *Error {
	return NewWithDetails(http.StatusBadRequest, code, message, details)
}

func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, "unauthorized", message)
}

func Forbidden(message string) *Error {
	return New(http.StatusForbidden, "forbidden", message)
}

func NotFound(resource string) *Error {
	return New(http.StatusNotFound, "not_found", resource+" not found")
}

func Conflict(code string, message string) *Error {
	return New(http.StatusConflict, code, message)
}

func PayloadTooLarge() *Error {
	return New(http.StatusRequestEntityTooLarge, "payload_too_large", "request body too large")
}

func TooManyRequests() *Error {
	return New(http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
}

func Internal(err error) *Error {
	return Wrap(err, http.StatusInternalServerError, "internal_error", "internal server error")
}

func From(err error) *Error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}

	return Internal(err)
}
