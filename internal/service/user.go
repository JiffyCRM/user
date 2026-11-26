package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	userv1 "github.com/JiffyCRM/user/api/user/v1"
	"github.com/JiffyCRM/user/internal/auth"
	"github.com/JiffyCRM/user/internal/store"
)

type UserService struct {
	userv1.UnimplementedUserServiceServer

	store  store.Store
	logger *slog.Logger
}

func New(st store.Store, logger *slog.Logger) *UserService {
	return &UserService{store: st, logger: logger}
}

func currentUserID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "x-user-id metadata is required")
	}
	return userID, nil
}

func (s *UserService) UpsertUser(ctx context.Context, req *userv1.UpsertUserRequest) (*userv1.User, error) {
	callerID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	internalID, err := parseUUIDField("internal_id", req.GetInternalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if internalID != callerID {
		return nil, status.Error(codes.PermissionDenied, "you can only manage your own profile")
	}
	name, err := normalizeRequiredName("name", req.GetName(), 100)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	organizationID, err := parseUUIDField("organization_id", req.GetOrganizationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, err := s.store.UpsertUser(ctx, store.UpsertUserInput{
		InternalID:     internalID,
		Name:           name,
		OrganizationID: organizationID,
	})
	if err != nil {
		return nil, s.mapStoreError("UpsertUser", err)
	}
	return userToProto(user), nil
}

func (s *UserService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.User, error) {
	if _, err := currentUserID(ctx); err != nil {
		return nil, err
	}
	internalID, err := parseUUIDField("internal_id", req.GetInternalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, err := s.store.GetUser(ctx, internalID)
	if err != nil {
		return nil, s.mapStoreError("GetUser", err)
	}
	return userToProto(user), nil
}

func (s *UserService) ListUsersByOrganization(ctx context.Context, req *userv1.ListUsersByOrganizationRequest) (*userv1.ListUsersResponse, error) {
	if _, err := currentUserID(ctx); err != nil {
		return nil, err
	}
	organizationID, err := parseUUIDField("organization_id", req.GetOrganizationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	users, err := s.store.ListUsersByOrganization(ctx, organizationID)
	if err != nil {
		return nil, s.mapStoreError("ListUsersByOrganization", err)
	}
	return &userv1.ListUsersResponse{Users: usersToProto(users)}, nil
}

func (s *UserService) ListUserMembers(ctx context.Context, req *userv1.ListUserMembersRequest) (*userv1.ListUserMembersResponse, error) {
	if _, err := currentUserID(ctx); err != nil {
		return nil, err
	}
	internalID, err := parseUUIDField("internal_id", req.GetInternalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	members, err := s.store.ListUserMembers(ctx, internalID)
	if err != nil {
		return nil, s.mapStoreError("ListUserMembers", err)
	}
	return &userv1.ListUserMembersResponse{UserMembers: userMembersToProto(members)}, nil
}

func (s *UserService) CreateUserMember(ctx context.Context, req *userv1.CreateUserMemberRequest) (*userv1.UserMember, error) {
	callerID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	internalID, err := parseUUIDField("internal_id", req.GetInternalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if internalID != callerID {
		return nil, status.Error(codes.PermissionDenied, "you can only manage your own roles")
	}
	roleID, err := parseUUIDField("role_id", req.GetRoleId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	member, err := s.store.CreateUserMember(ctx, internalID, roleID)
	if err != nil {
		return nil, s.mapStoreError("CreateUserMember", err)
	}
	return userMemberToProto(member), nil
}

func (s *UserService) DeleteUserMember(ctx context.Context, req *userv1.DeleteUserMemberRequest) (*emptypb.Empty, error) {
	callerID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	internalID, err := parseUUIDField("internal_id", req.GetInternalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if internalID != callerID {
		return nil, status.Error(codes.PermissionDenied, "you can only manage your own roles")
	}
	roleID, err := parseUUIDField("role_id", req.GetRoleId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.store.DeleteUserMember(ctx, internalID, roleID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user member not found")
		}
		return nil, s.mapStoreError("DeleteUserMember", err)
	}
	return &emptypb.Empty{}, nil
}
