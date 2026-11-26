package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	userv1 "github.com/JiffyCRM/user/api/user/v1"
	"github.com/JiffyCRM/user/internal/auth"
	"github.com/JiffyCRM/user/internal/store"
)

type fakeStore struct {
	store.Store

	upsertUserFn       func(context.Context, store.UpsertUserInput) (store.User, error)
	getUserFn          func(context.Context, uuid.UUID) (store.User, error)
	listUsersByOrgFn   func(context.Context, uuid.UUID) ([]store.User, error)
	listUserMembersFn  func(context.Context, uuid.UUID) ([]store.UserMember, error)
	createUserMemberFn func(context.Context, uuid.UUID, uuid.UUID) (store.UserMember, error)
	deleteUserMemberFn func(context.Context, uuid.UUID, uuid.UUID) error
}

func (f *fakeStore) UpsertUser(ctx context.Context, input store.UpsertUserInput) (store.User, error) {
	if f.upsertUserFn != nil {
		return f.upsertUserFn(ctx, input)
	}
	return store.User{}, store.ErrNotFound
}

func (f *fakeStore) GetUser(ctx context.Context, internalID uuid.UUID) (store.User, error) {
	if f.getUserFn != nil {
		return f.getUserFn(ctx, internalID)
	}
	return store.User{}, store.ErrNotFound
}

func (f *fakeStore) ListUsersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]store.User, error) {
	if f.listUsersByOrgFn != nil {
		return f.listUsersByOrgFn(ctx, organizationID)
	}
	return nil, store.ErrNotFound
}

func (f *fakeStore) ListUserMembers(ctx context.Context, internalID uuid.UUID) ([]store.UserMember, error) {
	if f.listUserMembersFn != nil {
		return f.listUserMembersFn(ctx, internalID)
	}
	return nil, store.ErrNotFound
}

func (f *fakeStore) CreateUserMember(ctx context.Context, internalID, roleID uuid.UUID) (store.UserMember, error) {
	if f.createUserMemberFn != nil {
		return f.createUserMemberFn(ctx, internalID, roleID)
	}
	return store.UserMember{}, store.ErrNotFound
}

func (f *fakeStore) DeleteUserMember(ctx context.Context, internalID, roleID uuid.UUID) error {
	if f.deleteUserMemberFn != nil {
		return f.deleteUserMemberFn(ctx, internalID, roleID)
	}
	return store.ErrNotFound
}

func (f *fakeStore) Ping(ctx context.Context) error { return nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, nil))
}

func contextForUser(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(auth.MetadataKey, userID.String()))
	interceptor := auth.UnaryServerInterceptor()
	var captured context.Context
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/user.v1.UserService/CreateUserMember",
	}, func(ctx context.Context, _ any) (any, error) {
		captured = ctx
		return nil, nil
	})
	if err != nil {
		t.Fatalf("auth interceptor error = %v", err)
	}
	return captured
}

func mustUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", s, err)
	}
	return id
}

func TestUpsertUser(t *testing.T) {
	callerID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	orgID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c8")

	st := &fakeStore{
		upsertUserFn: func(ctx context.Context, input store.UpsertUserInput) (store.User, error) {
			if input.InternalID != callerID {
				t.Errorf("internal_id = %v, want %v", input.InternalID, callerID)
			}
			if input.Name != "Alice" {
				t.Errorf("name = %q, want %q", input.Name, "Alice")
			}
			if input.OrganizationID != orgID {
				t.Errorf("organization_id = %v, want %v", input.OrganizationID, orgID)
			}
			return store.User{
				InternalID:     callerID,
				Name:           "Alice",
				OrganizationID: orgID,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}, nil
		},
	}

	svc := New(st, testLogger())
	ctx := contextForUser(t, callerID)

	user, err := svc.UpsertUser(ctx, &userv1.UpsertUserRequest{
		InternalId:     callerID.String(),
		Name:           "Alice",
		OrganizationId: orgID.String(),
	})
	if err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}
	if user.InternalId != callerID.String() {
		t.Errorf("InternalId = %q, want %q", user.InternalId, callerID.String())
	}
	if user.Name != "Alice" {
		t.Errorf("Name = %q, want %q", user.Name, "Alice")
	}
	if user.OrganizationId != orgID.String() {
		t.Errorf("OrganizationId = %q, want %q", user.OrganizationId, orgID.String())
	}
}

func TestUpsertUserWrongCaller(t *testing.T) {
	callerID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	otherID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c9")

	st := &fakeStore{}
	svc := New(st, testLogger())
	ctx := contextForUser(t, callerID)

	_, err := svc.UpsertUser(ctx, &userv1.UpsertUserRequest{
		InternalId:     otherID.String(),
		Name:           "Alice",
		OrganizationId: "6ba7b811-9dad-11d1-80b4-00c04fd430c8",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if code := status.Code(err); code != codes.PermissionDenied {
		t.Errorf("code = %v, want %v", code, codes.PermissionDenied)
	}
}

func TestCreateUserMember(t *testing.T) {
	callerID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	roleID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c9")

	st := &fakeStore{
		createUserMemberFn: func(ctx context.Context, internalID, roleID uuid.UUID) (store.UserMember, error) {
			if internalID != callerID {
				t.Errorf("internal_id = %v, want %v", internalID, callerID)
			}
			if roleID != roleID {
				t.Errorf("role_id = %v, want %v", roleID, roleID)
			}
			return store.UserMember{
				InternalID: callerID,
				RoleID:     roleID,
				AssignedAt: time.Now(),
			}, nil
		},
	}

	svc := New(st, testLogger())
	ctx := contextForUser(t, callerID)

	member, err := svc.CreateUserMember(ctx, &userv1.CreateUserMemberRequest{
		InternalId: callerID.String(),
		RoleId:     roleID.String(),
	})
	if err != nil {
		t.Fatalf("CreateUserMember() error = %v", err)
	}
	if member.InternalId != callerID.String() {
		t.Errorf("InternalId = %q, want %q", member.InternalId, callerID.String())
	}
	if member.RoleId != roleID.String() {
		t.Errorf("RoleId = %q, want %q", member.RoleId, roleID.String())
	}
	if member.AssignedAt == nil {
		t.Fatal("AssignedAt is nil")
	}
}

func TestCreateUserMemberWrongCaller(t *testing.T) {
	callerID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	otherID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c9")
	roleID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430ca")

	st := &fakeStore{}
	svc := New(st, testLogger())
	ctx := contextForUser(t, callerID)

	_, err := svc.CreateUserMember(ctx, &userv1.CreateUserMemberRequest{
		InternalId: otherID.String(),
		RoleId:     roleID.String(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if code := status.Code(err); code != codes.PermissionDenied {
		t.Errorf("code = %v, want %v", code, codes.PermissionDenied)
	}
}

func TestDeleteUserMember(t *testing.T) {
	callerID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	roleID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c9")

	st := &fakeStore{
		deleteUserMemberFn: func(ctx context.Context, internalID, roleID uuid.UUID) error {
			if internalID != callerID {
				t.Errorf("internal_id = %v, want %v", internalID, callerID)
			}
			if roleID != roleID {
				t.Errorf("role_id = %v, want %v", roleID, roleID)
			}
			return nil
		},
	}

	svc := New(st, testLogger())
	ctx := contextForUser(t, callerID)

	if _, err := svc.DeleteUserMember(ctx, &userv1.DeleteUserMemberRequest{
		InternalId: callerID.String(),
		RoleId:     roleID.String(),
	}); err != nil {
		t.Fatalf("DeleteUserMember() error = %v", err)
	}
}

func TestListUserMembers(t *testing.T) {
	callerID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	roleID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c9")

	st := &fakeStore{
		listUserMembersFn: func(ctx context.Context, internalID uuid.UUID) ([]store.UserMember, error) {
			if internalID != callerID {
				t.Errorf("internal_id = %v, want %v", internalID, callerID)
			}
			return []store.UserMember{
				{InternalID: callerID, RoleID: roleID, AssignedAt: time.Now()},
			}, nil
		},
	}

	svc := New(st, testLogger())
	ctx := contextForUser(t, callerID)

	resp, err := svc.ListUserMembers(ctx, &userv1.ListUserMembersRequest{
		InternalId: callerID.String(),
	})
	if err != nil {
		t.Fatalf("ListUserMembers() error = %v", err)
	}
	if len(resp.UserMembers) != 1 {
		t.Fatalf("len(UserMembers) = %d, want 1", len(resp.UserMembers))
	}
	if resp.UserMembers[0].RoleId != roleID.String() {
		t.Errorf("RoleId = %q, want %q", resp.UserMembers[0].RoleId, roleID.String())
	}
}

func TestGetUser(t *testing.T) {
	callerID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	st := &fakeStore{
		getUserFn: func(ctx context.Context, internalID uuid.UUID) (store.User, error) {
			if internalID != callerID {
				t.Errorf("internal_id = %v, want %v", internalID, callerID)
			}
			return store.User{
				InternalID:     callerID,
				Name:           "Alice",
				OrganizationID: mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c8"),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}, nil
		},
	}

	svc := New(st, testLogger())
	ctx := contextForUser(t, callerID)

	user, err := svc.GetUser(ctx, &userv1.GetUserRequest{
		InternalId: callerID.String(),
	})
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if user.InternalId != callerID.String() {
		t.Errorf("InternalId = %q, want %q", user.InternalId, callerID.String())
	}
	if user.Name != "Alice" {
		t.Errorf("Name = %q, want %q", user.Name, "Alice")
	}
}

func TestListUsersByOrganization(t *testing.T) {
	orgID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c8")
	callerID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	st := &fakeStore{
		listUsersByOrgFn: func(ctx context.Context, organizationID uuid.UUID) ([]store.User, error) {
			if organizationID != orgID {
				t.Errorf("organization_id = %v, want %v", organizationID, orgID)
			}
			return []store.User{
				{InternalID: callerID, Name: "Alice", OrganizationID: orgID},
			}, nil
		},
	}

	svc := New(st, testLogger())
	ctx := contextForUser(t, callerID)

	resp, err := svc.ListUsersByOrganization(ctx, &userv1.ListUsersByOrganizationRequest{
		OrganizationId: orgID.String(),
	})
	if err != nil {
		t.Fatalf("ListUsersByOrganization() error = %v", err)
	}
	if len(resp.Users) != 1 {
		t.Fatalf("len(Users) = %d, want 1", len(resp.Users))
	}
	if resp.Users[0].Name != "Alice" {
		t.Errorf("Name = %q, want %q", resp.Users[0].Name, "Alice")
	}
}
