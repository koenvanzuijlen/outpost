package server

import "context"

// sourceURL is the location of the Outpost source code. The footer links to it. A modified version of Outpost sets
// sourceURL to its own source code, as the AGPL requires.
const sourceURL = "https://github.com/koenvanzuijlen/outpost"

// igdbURL is the location of IGDB, the source of the game data. The footer links to it.
const igdbURL = "https://www.igdb.com/"

// versionContextKey is the context key for the version of Outpost. Its type is unexported, so no other package can
// use the same key.
type versionContextKey struct{}

// withVersion returns a copy of ctx that holds version.
func withVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, versionContextKey{}, version)
}

// versionFromContext returns the version that withVersion put in ctx. It returns "" if ctx holds no version.
func versionFromContext(ctx context.Context) string {
	v, _ := ctx.Value(versionContextKey{}).(string)
	return v
}
