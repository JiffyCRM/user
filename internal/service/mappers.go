package service

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	userv1 "github.com/JiffyCRM/user/api/user/v1"
	"github.com/JiffyCRM/user/internal/store"
)

func userToProto(user store.User) *userv1.User {
	return &userv1.User{
		InternalId:     user.InternalID.String(),
		Name:           user.Name,
		OrganizationId: user.OrganizationID.String(),
		CreatedAt:      timestamppb.New(user.CreatedAt),
		UpdatedAt:      timestamppb.New(user.UpdatedAt),
	}
}

func usersToProto(users []store.User) []*userv1.User {
	result := make([]*userv1.User, 0, len(users))
	for _, user := range users {
		result = append(result, userToProto(user))
	}
	return result
}

func userMemberToProto(member store.UserMember) *userv1.UserMember {
	return &userv1.UserMember{
		InternalId: member.InternalID.String(),
		RoleId:     member.RoleID.String(),
		AssignedAt: timestamppb.New(member.AssignedAt),
	}
}

func userMembersToProto(members []store.UserMember) []*userv1.UserMember {
	result := make([]*userv1.UserMember, 0, len(members))
	for _, member := range members {
		result = append(result, userMemberToProto(member))
	}
	return result
}