package store

import "errors"

// ErrNotFound is returned when a row lookup/update affects no rows.
var ErrNotFound = errors.New("record not found")
