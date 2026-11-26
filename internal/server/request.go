package server

import (
	"net/http"
	"strconv"
	"time"
)

// pathID returns path value name as a positive int64. ok is false if it is missing, not a number, or not positive.
func pathID(r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

// formOptionalPositiveInt returns form value name from the POST body as a positive int64. An empty value gives nil and
// true; a value that isn't a positive integer gives nil and false.
func formOptionalPositiveInt(r *http.Request, name string) (*int64, bool) {
	formValue := r.PostFormValue(name)
	if formValue == "" {
		return nil, true
	}

	parsed, err := strconv.ParseInt(formValue, 10, 64)
	if err != nil || parsed <= 0 {
		return nil, false
	}
	return &parsed, true
}

// formPositiveInts returns every form value name from the POST body as a positive int64, in the order of the body. No
// values give an empty slice and true. A value that isn't a positive integer, or a body that doesn't parse, gives nil
// and false.
func formPositiveInts(r *http.Request, name string) ([]int64, bool) {
	err := r.ParseForm()
	if err != nil {
		return nil, false
	}

	result := make([]int64, 0, len(r.PostForm[name]))
	for _, v := range r.PostForm[name] {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil || parsed <= 0 {
			return nil, false
		}
		result = append(result, parsed)
	}
	return result, true
}

// formOptionalDate returns form value name from the POST body as a date (2006-01-02, midnight UTC). An empty value
// gives nil and true; a value that isn't a valid date gives nil and false.
func formOptionalDate(r *http.Request, name string) (*time.Time, bool) {
	formValue := r.PostFormValue(name)
	if formValue == "" {
		return nil, true
	}

	parsed, err := time.Parse(time.DateOnly, formValue)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}
