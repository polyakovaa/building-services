package clients

import (
	projectv1 "building-services/gen/project/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProjectClient struct {
	Project    projectv1.ProjectServiceClient
	Member     projectv1.ProjectMemberServiceClient
	Task       projectv1.TaskServiceClient
	Timeline   projectv1.ProjectTimelineServiceClient
	Attachment projectv1.AttachmentServiceClient
	Conn       *grpc.ClientConn
}

func NewProjectClient(address string) (*ProjectClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	return &ProjectClient{
		Project:    projectv1.NewProjectServiceClient(conn),
		Member:     projectv1.NewProjectMemberServiceClient(conn),
		Task:       projectv1.NewTaskServiceClient(conn),
		Timeline:   projectv1.NewProjectTimelineServiceClient(conn),
		Attachment: projectv1.NewAttachmentServiceClient(conn),
		Conn:       conn,
	}, conn, nil
}
