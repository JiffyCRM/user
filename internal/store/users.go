package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *Postgres) UpsertUser(ctx context.Context, input UpsertUserInput) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, wrap("begin upsert user transaction", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, query("organization_upsert"), input.OrganizationID); err != nil {
		return User{}, wrap("upsert organization", err)
	}

	user, err := scanUser(tx.QueryRow(ctx, query("user_upsert"), input.InternalID, input.Name, input.OrganizationID))
	if err != nil {
		return User{}, wrap("upsert user", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, wrap("commit upsert user transaction", err)
	}
	return user, nil
}

func (s *Postgres) GetUser(ctx context.Context, internalID uuid.UUID) (User, error) {
	return scanUser(s.pool.QueryRow(ctx, query("user_get"), internalID))
}

func (s *Postgres) ListUsersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]User, error) {
	rows, err := s.pool.Query(ctx, query("user_list_by_organization"), organizationID)
	if err != nil {
		return nil, wrap("query users", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, wrap("iterate users", err)
	}
	return users, nil
}

func (s *Postgres) ListUserMembers(ctx context.Context, internalID uuid.UUID) ([]UserMember, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, query("user_exists"), internalID).Scan(&exists); err != nil {
		return nil, wrap("check user exists", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := s.pool.Query(ctx, query("user_member_list"), internalID)
	if err != nil {
		return nil, wrap("query user members", err)
	}
	defer rows.Close()

	members := make([]UserMember, 0)
	for rows.Next() {
		var member UserMember
		if err := rows.Scan(&member.InternalID, &member.RoleID, &member.AssignedAt); err != nil {
			return nil, wrap("scan user member", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, wrap("iterate user members", err)
	}
	return members, nil
}

func (s *Postgres) DeleteUserMember(ctx context.Context, internalID, roleID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, query("user_member_delete"), internalID, roleID)
	if err != nil {
		return wrap("delete user member", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Postgres) CreateUserMember(ctx context.Context, internalID, roleID uuid.UUID) (UserMember, error) {
	var member UserMember
	if err := s.pool.QueryRow(ctx, query("user_member_create"), internalID, roleID).Scan(
		&member.InternalID,
		&member.RoleID,
		&member.AssignedAt,
	); err != nil {
		return UserMember{}, wrap("create user member", err)
	}
	return member, nil
}
