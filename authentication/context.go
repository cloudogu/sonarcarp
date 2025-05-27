package authentication

import "context"

type authenticationContextKey int

const (
	unauthenticatedKey authenticationContextKey = iota
)

func WithUnauthenticatedRequest(ctx context.Context) context.Context {
	return context.WithValue(ctx, unauthenticatedKey, struct{}{})
}

func UnauthenticatedRequestAllowed(ctx context.Context) bool {
	allowed := ctx.Value(unauthenticatedKey)
	if allowed == nil {
		return false
	}

	return true
}
