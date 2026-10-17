package server

import (
	"net/http"
	"strconv"
)

// pathID extracts an integer ID from the URL path parameters. It returns the ID and a boolean indicating success.
func pathID(r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

// formOptionalPositiveInt extracts an optional positive integer from the form values. It returns a pointer to the
// integer and a boolean indicating success. An empty form value is considered valid and results in nil and true. If the
// form value can't be parsed as a positive integer, it returns nil and false.
func formOptionalPositiveInt(r *http.Request, name string) (*int64, bool) {
	formValue := r.PostFormValue(name)
	var value *int64
	if formValue != "" {
		parsed, err := strconv.ParseInt(formValue, 10, 64)
		if err != nil || parsed <= 0 {
			return nil, false
		}
		value = &parsed
	}
	return value, true
}
