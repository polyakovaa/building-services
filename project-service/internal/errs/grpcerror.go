package errs

import (
	"errors"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrNoPermission    = errors.New("permission denied")
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrUserNotFound    = errors.New("user not found")
	ErrProjectNotFound = errors.New("project not found")
	ErrMemberNotFound  = errors.New("member not found")
	ErrTaskNotFound    = errors.New("task not found")
)

func Handle(err error, method string) error {
	log.Printf("[ERROR] %s failed: %v", method, err)

	switch {
	case errors.Is(err, ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrNoPermission):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrProjectNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrMemberNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrTaskNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
