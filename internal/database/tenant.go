package database

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey int

const (
	ctxKeyUser ctxKey = iota
	ctxKeyOrg
)

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyUser, id)
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKeyUser).(uuid.UUID)
	return v, ok
}

func WithOrgID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyOrg, id)
}

// TenantScoped returns the organization_id for the current request, or
// (uuid.Nil, false) when none is set. Repositories that touch tenant
// tables MUST call this and inject the filter into every query.
func TenantScoped(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKeyOrg).(uuid.UUID)
	return v, ok
}
