package middleware

import (
	"context"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        expectedKey := os.Getenv("GRPC_API_KEY")
        if expectedKey == "" {
            // Only for study
            return handler(ctx, req)
        }

        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "Missing metadata")
        }

        values := md.Get("authorization")
        if len(values) == 0 {
            values = md.Get("x-api-key")
        }
        if len(values) == 0 {
            return nil, status.Error(codes.Unauthenticated, "Missing authorization")
        }

        apiKey := values[0]
        if apiKey != expectedKey {
            return nil, status.Error(codes.Unauthenticated, "Invalid authorization")
        }

        return handler(ctx, req)
    }
}
