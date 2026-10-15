package database

import "errors"

// ErrPlatformNotFound is returned when a platform is not found in the database.
var ErrPlatformNotFound = errors.New("platform not found")
