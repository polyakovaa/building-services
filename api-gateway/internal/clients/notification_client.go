package clients

import (
	notificationv1 "building-services/gen/notification/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewNotificationClient(address string) (notificationv1.NotificationServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return notificationv1.NewNotificationServiceClient(conn), conn, nil
}
