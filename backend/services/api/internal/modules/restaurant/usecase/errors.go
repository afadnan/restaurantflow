package usecase

import "errors"

var (
	ErrUseCaseNotConfigured = errors.New("restaurant use case is not configured")
	ErrInvalidPagination    = errors.New("invalid pagination")
)
