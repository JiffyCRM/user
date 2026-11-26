package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/JiffyCRM/user/internal/store"
)

func mustUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", s, err)
	}
	return id
}

func TestUpsertUserAndGetUser(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	st := store.NewPostgres(pool)
	orgID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c8")
	internalID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	insertOrganization(ctx, pool, orgID)

	user, err := st.UpsertUser(ctx, store.UpsertUserInput{
		InternalID:     internalID,
		Name:           "Alice",
		OrganizationID: orgID,
	})
	require.NoError(t, err)
	require.Equal(t, internalID, user.InternalID)
	require.Equal(t, "Alice", user.Name)
	require.Equal(t, orgID, user.OrganizationID)

	got, err := st.GetUser(ctx, internalID)
	require.NoError(t, err)
	require.Equal(t, internalID, got.InternalID)
	require.Equal(t, "Alice", got.Name)
	require.Equal(t, orgID, got.OrganizationID)
}

func TestUpsertUserUpdatesExisting(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	st := store.NewPostgres(pool)
	orgID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c8")
	orgID2 := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c9")
	internalID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	insertOrganization(ctx, pool, orgID)
	insertOrganization(ctx, pool, orgID2)

	_, err := st.UpsertUser(ctx, store.UpsertUserInput{
		InternalID:     internalID,
		Name:           "Alice",
		OrganizationID: orgID,
	})
	require.NoError(t, err)

	user, err := st.UpsertUser(ctx, store.UpsertUserInput{
		InternalID:     internalID,
		Name:           "Bob",
		OrganizationID: orgID2,
	})
	require.NoError(t, err)
	require.Equal(t, "Bob", user.Name)
	require.Equal(t, orgID2, user.OrganizationID)
}

func TestGetUserNotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	st := store.NewPostgres(pool)
	_, err := st.GetUser(ctx, uuid.New())
	require.ErrorIs(t, err, store.ErrNotFound)
}

func TestListUsersByOrganization(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	st := store.NewPostgres(pool)
	orgID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c8")
	orgID2 := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c9")
	internalID1 := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	internalID2 := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c9")

	insertOrganization(ctx, pool, orgID)
	insertOrganization(ctx, pool, orgID2)

	_, err := st.UpsertUser(ctx, store.UpsertUserInput{
		InternalID:     internalID1,
		Name:           "Alice",
		OrganizationID: orgID,
	})
	require.NoError(t, err)
	_, err = st.UpsertUser(ctx, store.UpsertUserInput{
		InternalID:     internalID2,
		Name:           "Bob",
		OrganizationID: orgID,
	})
	require.NoError(t, err)
	_, err = st.UpsertUser(ctx, store.UpsertUserInput{
		InternalID:     uuid.New(),
		Name:           "Carol",
		OrganizationID: orgID2,
	})
	require.NoError(t, err)

	users, err := st.ListUsersByOrganization(ctx, orgID)
	require.NoError(t, err)
	require.Len(t, users, 2)

	users2, err := st.ListUsersByOrganization(ctx, orgID2)
	require.NoError(t, err)
	require.Len(t, users2, 1)
}

func TestCreateAndListUserMembers(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	st := store.NewPostgres(pool)
	orgID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c8")
	internalID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	roleID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c9")
	roleID2 := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430ca")

	insertOrganization(ctx, pool, orgID)
	_, err := st.UpsertUser(ctx, store.UpsertUserInput{
		InternalID:     internalID,
		Name:           "Alice",
		OrganizationID: orgID,
	})
	require.NoError(t, err)

	member, err := st.CreateUserMember(ctx, internalID, roleID)
	require.NoError(t, err)
	require.Equal(t, internalID, member.InternalID)
	require.Equal(t, roleID, member.RoleID)
	require.NotZero(t, member.AssignedAt)

	member2, err := st.CreateUserMember(ctx, internalID, roleID2)
	require.NoError(t, err)
	require.Equal(t, roleID2, member2.RoleID)

	members, err := st.ListUserMembers(ctx, internalID)
	require.NoError(t, err)
	require.Len(t, members, 2)
}

func TestDeleteUserMember(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	st := store.NewPostgres(pool)
	orgID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c8")
	internalID := mustUUID(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	roleID := mustUUID(t, "6ba7b811-9dad-11d1-80b4-00c04fd430c9")

	insertOrganization(ctx, pool, orgID)
	_, err := st.UpsertUser(ctx, store.UpsertUserInput{
		InternalID:     internalID,
		Name:           "Alice",
		OrganizationID: orgID,
	})
	require.NoError(t, err)
	_, err = st.CreateUserMember(ctx, internalID, roleID)
	require.NoError(t, err)

	require.NoError(t, st.DeleteUserMember(ctx, internalID, roleID))

	members, err := st.ListUserMembers(ctx, internalID)
	require.NoError(t, err)
	require.Empty(t, members)

	err = st.DeleteUserMember(ctx, internalID, roleID)
	require.ErrorIs(t, err, store.ErrNotFound)
}

func TestCreateUserMemberUserNotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	st := store.NewPostgres(pool)
	_, err := st.CreateUserMember(ctx, uuid.New(), uuid.New())
	require.Error(t, err)
}

var _ = time.Time{}
