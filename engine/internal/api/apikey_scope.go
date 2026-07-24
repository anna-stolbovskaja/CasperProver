package api

import "context"

// keyScopeCtxKey is the context key writeAuth uses to stash the scope
// of the credential the request came in with. Private unexported type
// avoids collisions with any other context.Value users.
type keyScopeCtxKey struct{}

// sharedKeyScope is the pseudo-scope assigned when the caller presented
// the raw shared API_KEY (as opposed to a per-wallet sk_live_ key).
// Treated as a super-scope by requireScope so an operator with the
// shared secret can hit every public write route. Not a valid value
// for isValidScope() and never persisted in api_keys.scope.
const sharedKeyScope = "__shared_api_key__"

func withKeyScope(ctx context.Context, scope string) context.Context {
	return context.WithValue(ctx, keyScopeCtxKey{}, scope)
}

func keyScopeFromContext(ctx context.Context) string {
	v, _ := ctx.Value(keyScopeCtxKey{}).(string)
	return v
}
