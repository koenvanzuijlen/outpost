package server

import (
	"net/http"
	"net/url"
	"strings"
)

// signInURL returns the path of the sign-in page for r. For a GET request to a page other than the home page, the
// path has the query value next with the path and the query of r, so the user returns to that page after sign-in.
// Other requests give "/sign-in".
func signInURL(r *http.Request) string {
	if r.Method != http.MethodGet {
		return "/sign-in"
	}

	requested := r.URL.RequestURI()
	if requested == "/" {
		return "/sign-in"
	}

	values := url.Values{}
	values.Set("next", requested)
	return "/sign-in?" + values.Encode()
}

// nextPath returns raw if it is a local path, and "/" otherwise. The sign-in pages redirect to the result, so a link
// cannot send the user to another site. A local path starts with one "/" that no "/" or "\" follows. It has no
// scheme, no host and no control characters.
func nextPath(raw string) string {
	if !strings.HasPrefix(raw, "/") {
		return "/"
	}
	if len(raw) > 1 && (raw[1] == '/' || raw[1] == '\\') {
		return "/"
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" {
		return "/"
	}

	return raw
}
