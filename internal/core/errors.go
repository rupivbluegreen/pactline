// Package core holds pactline's domain types and the base error type.
//
// Domain entities (Organization, Contract, ContractDocument, etc.) land
// here in Phase 1; this file establishes the error contract.
package core

import "fmt"

// Error is the base type for pactline domain errors. Concrete errors
// embed *Error and name the failed precondition. Use fmt.Errorf for
// non-domain wrapping.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
