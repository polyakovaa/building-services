package clients

import (
	projectv1 "building-services/gen/project/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProjectClient struct {
	Project       projectv1.ProjectServiceClient
	ProjectMember projectv1.ProjectMemberServiceClient
	Task          projectv1.TaskServiceClient
	Timeline      projectv1.ProjectTimelineServiceClient
	Attachment    projectv1.AttachmentServiceClient
	User          projectv1.ProjectServiceClient
	Department    projectv1.DepartmentServiceClient
	Conn          *grpc.ClientConn
}

func NewProjectClient(address string) (*ProjectClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	return &ProjectClient{
		Project:       projectv1.NewProjectServiceClient(conn),
		ProjectMember: projectv1.NewProjectMemberServiceClient(conn),
		Task:          projectv1.NewTaskServiceClient(conn),
		Timeline:      projectv1.NewProjectTimelineServiceClient(conn),
		Attachment:    projectv1.NewAttachmentServiceClient(conn),
		User:          projectv1.NewProjectServiceClient(conn),
		Department:    projectv1.NewDepartmentServiceClient(conn),
		Conn:          conn,
	}, conn, nil
}
