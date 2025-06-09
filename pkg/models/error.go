package models

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

const (
	SessionNotCreatedErr    = "session not created"
	BadSessionParametersErr = "bad session parameters"

	// StatusRequestCancelled unofficial status code, actually it won't be sent over the wire, we just need a marker
	StatusRequestCancelled = 499
)

type ErrorWithCode interface {
	error
	Code() int
}

type ErrorMessage struct {
	code    int
	err     error
	Message string `json:"message"`
}

func NewErrorMessage(code int, err error) *ErrorMessage {
	return &ErrorMessage{
		code:    code,
		err:     err,
		Message: err.Error(),
	}
}

func (e *ErrorMessage) Code() int {
	return e.code
}

func (e *ErrorMessage) Error() string {
	return e.err.Error()
}

func (e *ErrorMessage) Unwrap() error {
	return e.err
}

func NewBadRequestError(err error) *ErrorMessage {
	return NewErrorMessage(http.StatusBadRequest, err)
}

func NewNotFoundError(err error) *ErrorMessage {
	return NewErrorMessage(http.StatusNotFound, err)
}

func NewQuoteExceededError(err error) *ErrorMessage {
	return NewErrorMessage(http.StatusTooManyRequests, err)
}

func NewServiceUnavailableError(err error) *ErrorMessage {
	return NewErrorMessage(http.StatusServiceUnavailable, err)
}

func NewInternalServerError(err error) *ErrorMessage {
	return NewErrorMessage(http.StatusInternalServerError, err)
}

func NewTimeoutError(err error) *ErrorMessage {
	return NewErrorMessage(http.StatusGatewayTimeout, err)
}

func NewCancelledError(err error) *ErrorMessage {
	return NewErrorMessage(StatusRequestCancelled, err)
}

func WrapTimeoutErr(err error, msg string) error {
	var e ErrorWithCode
	if errors.Is(err, context.DeadlineExceeded) && !errors.As(err, &e) {
		err = NewTimeoutError(err)
	}
	return errors.Wrap(err, msg)
}

func WrapCancelledErr(err error) error {
	var e ErrorWithCode
	if errors.Is(err, context.Canceled) && !errors.As(err, &e) {
		err = NewCancelledError(err)
	}
	return err
}

type W3CError struct {
	code  int
	err   error
	Value ErrorBody `json:"value"`
}

func (w *W3CError) Error() string {
	return w.err.Error()
}

func (w *W3CError) Code() int {
	return w.code
}

func (w *W3CError) Unwrap() error {
	return w.err
}

type ErrorBody struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	StackTrace string `json:"stacktrace"`
}

func NewW3CErr(code int, errText string, err error) *W3CError {
	return &W3CError{
		code: code,
		err:  err,
		Value: ErrorBody{
			Error:      errText,
			Message:    err.Error(),
			StackTrace: fmt.Sprintf("%+v", err),
		},
	}
}

func BadWDSessionParameters(err error) *W3CError {
	return NewW3CErr(http.StatusBadRequest, BadSessionParametersErr, err)
}

func WDSessionNotCreatedError(err error) *W3CError {
	code := http.StatusInternalServerError
	var e ErrorWithCode
	if errors.As(err, &e) {
		code = e.Code()
	}
	return NewW3CErr(code, SessionNotCreatedErr, err)
}
