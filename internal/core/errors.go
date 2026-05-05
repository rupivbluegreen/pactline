// Package core holds pactline's domain types and the base error type.
package core

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")
	ErrTokenConsumed   = errors.New("token already consumed")
	ErrSessionRevoked  = errors.New("session revoked")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrNoOrganization  = errors.New("no organization context")
)
