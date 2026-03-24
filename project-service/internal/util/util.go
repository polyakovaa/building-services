package util

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func GetFromContext(ctx context.Context, meta string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("no metadata found")
	}

	values := md.Get(meta)
	if len(values) == 0 {
		return "", fmt.Errorf("%s not found in metadata", meta)
	}

	return values[0], nil
}

func NonEmpty(new, old string) string {
	if new != "" {
		return new
	}
	return old
}

func FirstNonNil(new, old *timestamppb.Timestamp) *timestamppb.Timestamp {
	if new != nil {
		return new
	}
	return old
}
