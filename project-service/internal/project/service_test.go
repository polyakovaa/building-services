package project

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/errs"
	"building-services/project-service/internal/user"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockProjectRepo struct {
	created []*projectv1.Project
	byID    map[string]*projectv1.Project
}

func (f *mockProjectRepo) Create(_ context.Context, project *projectv1.Project) error {
	project.Id = "project-1"
	f.created = append(f.created, project)
	if f.byID != nil {
		f.byID[project.Id] = project
	}
	return nil
}

func (f *mockProjectRepo) FindByID(_ context.Context, id string) (*projectv1.Project, error) {
	if f.byID != nil {
		if project, ok := f.byID[id]; ok {
			return project, nil
		}
	}
	return nil, sql.ErrNoRows
}
func (f *mockProjectRepo) Update(context.Context, *projectv1.Project) error { return nil }
func (f *mockProjectRepo) Delete(context.Context, string) error{ return nil }
func (f *mockProjectRepo) List(context.Context, *ProjectFilter) ([]*projectv1.Project, error) {
	return nil, nil
}
func (f *mockProjectRepo) UpdateStatus(context.Context, string, projectv1.ProjectStatus) error {
	return nil
}

type mockMemberRepo struct {
	added []*projectv1.ProjectMember
}

func (f *mockMemberRepo) Add(_ context.Context, member *projectv1.ProjectMember) error {
	f.added = append(f.added, member)
	return nil
}

type mockTimelineRepo struct {
	created []string
}

func (f *mockTimelineRepo) CreateEmpty(_ context.Context, projectID string) error {
	f.created = append(f.created, projectID)
	return nil
}

type mockUserRepo struct{}

func (mockUserRepo) FindByID(context.Context, string) (*user.User, error) { return nil, nil }
func (mockUserRepo) FindByEmail(context.Context, string) (*user.User, error) {
	return nil, nil
}
func (mockUserRepo) Find(context.Context, string, int) ([]*user.User, error) {
	return nil, nil
}

type mockPermissionChecker struct {
	allow bool
}

func (m mockPermissionChecker) Check(context.Context, string, string, string, string) (bool, error) {
	return m.allow, nil
}

type mockEventPublisher struct {
	events []publishedEvent
	err    error
}

type publishedEvent struct {
	routingKey string
	payload    map[string]interface{}
}

func (f *mockEventPublisher) Publish(_ context.Context, routingKey string, event map[string]interface{}) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, publishedEvent{routingKey: routingKey, payload: event})
	return nil
}

func (f *mockEventPublisher) Close() error { return nil }

type testServiceDeps struct {
	projectRepo  *mockProjectRepo
	memberRepo   *mockMemberRepo
	timelineRepo *mockTimelineRepo
	publisher    *mockEventPublisher
	authz        mockPermissionChecker
}

func validCreateProjectRequest() *projectv1.CreateProjectRequest {
	return &projectv1.CreateProjectRequest{
		Name:          "Test Project",
		Customer:      "Test Customer",
		ObjectAddress: "Address",
		StartDate:     timestamppb.New(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
	}
}

func newTestService(opts ...func(*testServiceDeps)) (*Service, context.Context, testServiceDeps) {
	deps := testServiceDeps{
		projectRepo:  &mockProjectRepo{},
		memberRepo:   &mockMemberRepo{},
		timelineRepo: &mockTimelineRepo{},
		publisher:    &mockEventPublisher{},
		authz:        mockPermissionChecker{allow: true},
	}
	for _, opt := range opts {
		opt(&deps)
	}

	svc := NewService(
		deps.projectRepo,
		deps.memberRepo,
		mockUserRepo{},
		deps.timelineRepo,
		deps.authz,
		deps.publisher,
	)
	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs("user_id", "user-1"),
	)
	return svc, ctx, deps
}

func TestService_CreateProject_Success(t *testing.T) {
	t.Parallel()

	svc, ctx, deps := newTestService()

	project, err := svc.CreateProject(ctx, validCreateProjectRequest())
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if project.CreatedBy != "user-1" {
		t.Fatalf("CreatedBy = %q, want %q", project.CreatedBy, "user-1")
	}
	if len(deps.projectRepo.created) != 1 {
		t.Fatalf("projectRepo.Create called %d times, want 1", len(deps.projectRepo.created))
	}
	if len(deps.memberRepo.added) != 1 {
		t.Fatalf("memberRepo.Add called %d times, want 1", len(deps.memberRepo.added))
	}
	if deps.memberRepo.added[0].UserId != "user-1" {
		t.Fatalf("member user = %q, want %q", deps.memberRepo.added[0].UserId, "user-1")
	}
	if len(deps.timelineRepo.created) != 1 || deps.timelineRepo.created[0] != "project-1" {
		t.Fatalf("timeline created for %v, want [project-1]", deps.timelineRepo.created)
	}
	if len(deps.publisher.events) != 1 {
		t.Fatalf("published %d events, want 1", len(deps.publisher.events))
	}
	ev := deps.publisher.events[0]
	if ev.routingKey != "project.created" {
		t.Fatalf("routing key = %q, want %q", ev.routingKey, "project.created")
	}
	if ev.payload["event_type"] != "project.created" {
		t.Fatalf("event_type = %v, want project.created", ev.payload["event_type"])
	}
	if ev.payload["actor_user_id"] != "user-1" {
		t.Fatalf("actor_user_id = %v, want user-1", ev.payload["actor_user_id"])
	}
	if ev.payload["project_id"] != "project-1" {
		t.Fatalf("project_id = %v, want project-1", ev.payload["project_id"])
	}
}

func TestService_CreateProject_InvalidInput(t *testing.T) {
	t.Parallel()

	svc, ctx, deps := newTestService()

	tests := []struct {
		name    string
		req     *projectv1.CreateProjectRequest
		wantErr error
	}{
		{
			name: "empty name",
			req: func() *projectv1.CreateProjectRequest {
				r := validCreateProjectRequest()
				r.Name = ""
				return r
			}(),
			wantErr: errs.ErrInvalidInput,
		},
		{
			name: "empty customer",
			req: func() *projectv1.CreateProjectRequest {
				r := validCreateProjectRequest()
				r.Customer = ""
				return r
			}(),
			wantErr: errs.ErrInvalidInput,
		},
		{
			name: "empty address",
			req: func() *projectv1.CreateProjectRequest {
				r := validCreateProjectRequest()
				r.ObjectAddress = ""
				return r
			}(),
			wantErr: errs.ErrInvalidInput,
		},
		{
			name: "empty start date",
			req: func() *projectv1.CreateProjectRequest {
				r := validCreateProjectRequest()
				r.StartDate = nil
				return r
			}(),
			wantErr: errs.ErrInvalidInput,
		},
		{
			name: "end date before start date",
			req: func() *projectv1.CreateProjectRequest {
				r := validCreateProjectRequest()
				r.EndDate = timestamppb.New(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
				return r
			}(),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.CreateProject(ctx, tt.req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("CreateProject() error = %v, want %v", err, tt.wantErr)
				}
			} else if err == nil {
				t.Fatalf("CreateProject() error = nil, want non-nil")
			}

			if len(deps.projectRepo.created) != 0 {
				t.Fatalf("projectRepo.Create called %d times, want 0", len(deps.projectRepo.created))
			}
		})
	}
}

func TestService_CreateProject_NoUserID(t *testing.T) {
	t.Parallel()

	svc, _, deps := newTestService()
	ctx := context.Background()

	_, err := svc.CreateProject(ctx, validCreateProjectRequest())
	if err == nil {
		t.Fatal("CreateProject() error = nil, want error")
	}
	if len(deps.projectRepo.created) != 0 {
		t.Fatalf("projectRepo.Create called %d times, want 0", len(deps.projectRepo.created))
	}
}

func TestService_CreateProject_PermissionDenied(t *testing.T) {
	t.Parallel()

	svc, ctx, deps := newTestService(func(d *testServiceDeps) {
		d.authz = mockPermissionChecker{allow: false}
	})

	_, err := svc.CreateProject(ctx, validCreateProjectRequest())
	if !errors.Is(err, errs.ErrNoPermission) {
		t.Fatalf("CreateProject() error = %v, want %v", err, errs.ErrNoPermission)
	}
	if len(deps.projectRepo.created) != 0 {
		t.Fatalf("projectRepo.Create called %d times, want 0", len(deps.projectRepo.created))
	}
}

func TestService_CreateProject_PublishFailureStillCreates(t *testing.T) {
	t.Parallel()

	svc, ctx, deps := newTestService(func(d *testServiceDeps) {
		d.publisher.err = errors.New("rabbit down")
	})

	project, err := svc.CreateProject(ctx, validCreateProjectRequest())
	if err != nil {
		t.Fatalf("CreateProject() error = %v, want nil", err)
	}
	if project.Id != "project-1" {
		t.Fatalf("project.Id = %q, want project-1", project.Id)
	}
	if len(deps.projectRepo.created) != 1 {
		t.Fatalf("projectRepo.Create called %d times, want 1", len(deps.projectRepo.created))
	}
}

func TestService_GetProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		req       *projectv1.GetProjectRequest
		allowAuth bool
		wantErr   error
	}{
		{
			name:      "success",
			req:       &projectv1.GetProjectRequest{Id: "project-1"},
			allowAuth: true,
			wantErr:   nil,
		},
		{
			name:      "not found",
			req:       &projectv1.GetProjectRequest{Id: "project-2"},
			allowAuth: true,
			wantErr:   errs.ErrProjectNotFound,
		},
		{
			name:      "invalid input",
			req:       &projectv1.GetProjectRequest{Id: ""},
			allowAuth: true,
			wantErr:   errs.ErrInvalidInput,
		},
		{
			name:      "permission denied",
			req:       &projectv1.GetProjectRequest{Id: "project-1"},
			allowAuth: false,
			wantErr:   errs.ErrNoPermission,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, ctx, _ := newTestService(func(d *testServiceDeps) {
				d.authz = mockPermissionChecker{allow: tt.allowAuth}
				d.projectRepo.byID = map[string]*projectv1.Project{
					"project-1": {
						Id:            "project-1",
						Name:          "Test Project",
						Customer:      "Test Customer",
						ObjectAddress: "Address",
					},
				}
			})

			project, err := svc.GetProject(ctx, tt.req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetProject() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetProject() error = %v, want nil", err)
			}
			if project == nil || project.Id != "project-1" {
				t.Fatalf("GetProject() = %+v, want project-1", project)
			}
		})
	}
}
