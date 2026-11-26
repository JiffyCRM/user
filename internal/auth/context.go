package auth

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const MetadataKey = "x-user-id"

const healthMethodPrefix = "/grpc.health.v1.Health/"

type userIDKey struct{}

// UserIDFromContext returns the authenticated internal user ID.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, false
	}
	return userID, true
}

// UnaryServerInterceptor extracts and validates the x-user-id metadata key.
// The standard gRPC health service is exempt so readiness probes work without
// an end-user identity.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, healthMethodPrefix) {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "x-user-id metadata is required")
		}

		values := md.Get(MetadataKey)
		if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
			return nil, status.Error(codes.Unauthenticated, "x-user-id metadata is required")
		}

		userID, err := uuid.Parse(strings.TrimSpace(values[0]))
		if err != nil || userID == uuid.Nil {
			return nil, status.Error(codes.Unauthenticated, "x-user-id metadata must be a valid UUID")
		}

		return handler(context.WithValue(ctx, userIDKey{}, userID), req)
	}
}