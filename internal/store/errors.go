package store

import "errors"

var (
	ErrNotFound      = errors.New("workload not found")
	ErrAlreadyExists = errors.New("workload already exists")
)
