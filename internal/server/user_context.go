package server

import (
	"context"
	"net/http"

	"github.com/koenvanzuijlen/outpost/internal/database"
)

// userContextKey is the context key of the signed-in user. The type is unexported, so no other package can use it.
type userContextKey struct{}

// withUser returns a copy of ctx that carries user.
func withUser(ctx context.Context, user database.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// userFromContext returns the user that withUser stored in ctx. It reports false if ctx has no user.
func userFromContext(ctx context.Context) (database.User, bool) {
	user, ok := ctx.Value(userContextKey{}).(database.User)
	return user, ok
}

// currentUser returns the signed-in user of the request. It panics if the request has no user, because only a route
// without requireUser can cause this.
func currentUser(r *http.Request) database.User {
	user, ok := userFromContext(r.Context())
	if !ok {
		panic("currentUser: no user in the request context; wrap the route with requireUser")
	}
	return user
}
