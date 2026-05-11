package clients

import (
	analyticsv1 "building-services/gen/analytics/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewAnalyticsClient(address string) (analyticsv1.AnalyticsServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return analyticsv1.NewAnalyticsServiceClient(conn), conn, nil
}
