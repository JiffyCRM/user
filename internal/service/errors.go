package service

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/JiffyCRM/user/internal/store"
)

func (s *UserService) mapStoreError(method string, err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, store.ErrConflict):
		return status.Error(codes.Aborted, "concurrent modification conflict, retry the request")
	default:
		s.logger.Error("user store error", "method", method, "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}