package domain

import "errors"

// ErrNotFound is returned by repositories when a lookup finds no row. It is
// a plain sentinel, not an *apperror.Error, so the domain/repository layers
// stay free of any HTTP concept — the service layer is what translates it
// into apperror.NotFound for a specific resource name.
var ErrNotFound = errors.New("domain: resource not found")

// ErrConflict is returned by repositories on a unique-constraint violation
// (e.g. a slug or email that already exists).
var ErrConflict = errors.New("domain: resource already exists")
