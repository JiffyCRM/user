package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("concurrent modification conflict")
)

type Organization struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	InternalID     uuid.UUID
	Name           string
	OrganizationID uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserMember struct {
	InternalID uuid.UUID
	RoleID     uuid.UUID
	AssignedAt time.Time
}

type UpsertUserInput struct {
	InternalID     uuid.UUID
	Name           string
	OrganizationID uuid.UUID
}

type Store interface {
	Ping(ctx context.Context) error

	UpsertUser(ctx context.Context, input UpsertUserInput) (User, error)
	GetUser(ctx context.Context, internalID uuid.UUID) (User, error)
	ListUsersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]User, error)

	ListUserMembers(ctx context.Context, internalID uuid.UUID) ([]UserMember, error)
	CreateUserMember(ctx context.Context, internalID, roleID uuid.UUID) (UserMember, error)
	DeleteUserMember(ctx context.Context, internalID, roleID uuid.UUID) error
}
