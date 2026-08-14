package model

import "errors"

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrDuplicate       = errors.New("duplicate resource")
)