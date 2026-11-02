package service

import "errors"

var (
	ErrNotFound   = errors.New("not_found")
	ErrForbidden  = errors.New("forbidden")
	ErrConflict   = errors.New("conflict")
	ErrValidation = errors.New("validation")
)

type AppError struct {
	Kind    error
	Message string
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Kind }

func notFound(msg string) error  { return &AppError{Kind: ErrNotFound, Message: msg} }
func forbidden(msg string) error { return &AppError{Kind: ErrForbidden, Message: msg} }
func conflict(msg string) error  { return &AppError{Kind: ErrConflict, Message: msg} }
func invalid(msg string) error   { return &AppError{Kind: ErrValidation, Message: msg} }
