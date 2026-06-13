package task

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

type mockTaskRepo struct {
	created []*projectv1.Task
	byID    map[string]*projectv1.Task
}

func (f *mockTaskRepo) Create(_ context.Context, task *projectv1.Task) error {
	task.Id = "task-1"
	f.created = append(f.created, task)
	if f.byID != nil {
		f.byID[task.Id] = task
	}
	return nil
}

func (f *mockTaskRepo) FindByID(_ context.Context, id string) (*projectv1.Task, error) {
	if f.byID == nil {
		return nil, sql.ErrNoRows
	}
	task, ok := f.byID[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	taskCopy := *task
	return &taskCopy, nil
}

func (f *mockTaskRepo) Update(context.Context, *projectv1.Task) error { return nil }
func (f *mockTaskRepo) Delete(context.Context, string) error           { return nil }
func (f *mockTaskRepo) List(context.Context, *TaskFilter) ([]*projectv1.Task, error) {
	return nil, nil
}
func (f *mockTaskRepo) UpdateStatus(_ context.Context, id string, status projectv1.TaskStatus, actualHours float64) error {
	if f.byID == nil {
		return sql.ErrNoRows
	}
	task, ok := f.byID[id]
	if !ok {
		return sql.ErrNoRows
	}
	task.Status = status
	if actualHours > 0 {
		task.ActualHours = actualHours
	}
	return nil
}
func (f *mockTaskRepo) UpdateLabor(context.Context, string, string, float64, float64) error {
	return nil
}
func (f *mockTaskRepo) Assign(_ context.Context, id string, assignedID string) (*projectv1.Task, error) {
	if f.byID == nil {
		return nil, sql.ErrNoRows
	}
	task, ok := f.byID[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	task.AssignedTo = assignedID
	taskCopy := *task
	return &taskCopy, nil
}

type mockProjectRepo struct {
	byID map[string]*projectv1.Project
}

func (f *mockProjectRepo) FindByID(_ context.Context, id string) (*projectv1.Project, error) {
	if f.byID != nil {
		if project, ok := f.byID[id]; ok {
			return project, nil
		}
	}
	return nil, sql.ErrNoRows
}

type mockUserRepo struct {
	users map[string]*user.User
}

func (m *mockUserRepo) FindByID(_ context.Context, id string) (*user.User, error) {
	if m.users != nil {
		if u, ok := m.users[id]; ok {
			return u, nil
		}
	}
	return &user.User{ID: id, FullName: "Test User", Email: "test@example.com"}, nil
}

type mockActivityRepo struct {
	exists map[string]bool
}

func (f *mockActivityRepo) Exists(_ context.Context, id string) (bool, error) {
	if f.exists == nil {
		return true, nil
	}
	return f.exists[id], nil
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
	taskRepo     *mockTaskRepo
	projectRepo  *mockProjectRepo
	userRepo     *mockUserRepo
	activityRepo *mockActivityRepo
	publisher    *mockEventPublisher
	authz        mockPermissionChecker
}

func validCreateTaskRequest() *projectv1.CreateTaskRequest {
	return &projectv1.CreateTaskRequest{
		ProjectId:      "project-1",
		Title:          "Test Task",
		Description:    "Test Description",
		ActivityTypeId: "activity-type-1",
		AssignedTo:       "user-1",
		Priority:       projectv1.TaskPriority_TASK_PRIORITY_LOW,
		Deadline:       timestamppb.New(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
		PlannedHours:   1.0,
	}
}

func newTestService(opts ...func(*testServiceDeps)) (*Service, context.Context, testServiceDeps) {
	deps := testServiceDeps{
		taskRepo:    &mockTaskRepo{},
		projectRepo: &mockProjectRepo{
			byID: map[string]*projectv1.Project{
				"project-1": {Id: "project-1", Name: "Test Project"},
			},
		},
		userRepo:     &mockUserRepo{},
		activityRepo: &mockActivityRepo{exists: map[string]bool{"activity-type-1": true}},
		publisher:    &mockEventPublisher{},
		authz:        mockPermissionChecker{allow: true},
	}
	for _, opt := range opts {
		opt(&deps)
	}

	svc := NewService(
		deps.taskRepo,
		deps.projectRepo,
		deps.userRepo,
		deps.activityRepo,
		deps.authz,
		deps.publisher,
	)
	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs("user_id", "user-1"),
	)
	return svc, ctx, deps
}

func withTaskInRepo(task *projectv1.Task) func(*testServiceDeps) {
	return func(d *testServiceDeps) {
		if d.taskRepo.byID == nil {
			d.taskRepo.byID = make(map[string]*projectv1.Task)
		}
		d.taskRepo.byID[task.Id] = task
	}
}

func TestService_CreateTask_Success(t *testing.T) {
	t.Parallel()

	svc, ctx, deps := newTestService()
	wantDeadline := timestamppb.New(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))

	task, err := svc.CreateTask(ctx, validCreateTaskRequest())
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if len(deps.taskRepo.created) != 1 {
		t.Fatalf("taskRepo.Create called %d times, want 1", len(deps.taskRepo.created))
	}
	if task.ProjectId != "project-1" {
		t.Fatalf("ProjectId = %q, want project-1", task.ProjectId)
	}
	if task.Title != "Test Task" {
		t.Fatalf("Title = %q, want Test Task", task.Title)
	}
	if task.Status != projectv1.TaskStatus_TASK_STATUS_TODO {
		t.Fatalf("Status = %v, want TODO", task.Status)
	}
	if task.Priority != projectv1.TaskPriority_TASK_PRIORITY_LOW {
		t.Fatalf("Priority = %v, want LOW", task.Priority)
	}
	if !task.Deadline.AsTime().Equal(wantDeadline.AsTime()) {
		t.Fatalf("Deadline = %v, want %v", task.Deadline.AsTime(), wantDeadline.AsTime())
	}
	if task.AssignedTo != "user-1" {
		t.Fatalf("AssignedTo = %q, want user-1", task.AssignedTo)
	}
	if task.CreatedBy != "user-1" {
		t.Fatalf("CreatedBy = %q, want user-1", task.CreatedBy)
	}
	if task.ActivityTypeId != "activity-type-1" {
		t.Fatalf("ActivityTypeId = %q, want activity-type-1", task.ActivityTypeId)
	}
	if task.PlannedHours != 1.0 {
		t.Fatalf("PlannedHours = %v, want 1.0", task.PlannedHours)
	}
	if len(deps.publisher.events) != 1 {
		t.Fatalf("published %d events, want 1", len(deps.publisher.events))
	}
	ev := deps.publisher.events[0]
	if ev.routingKey != "task.created" {
		t.Fatalf("routing key = %q, want task.created", ev.routingKey)
	}
	if ev.payload["event_type"] != "task.created" {
		t.Fatalf("event_type = %v, want task.created", ev.payload["event_type"])
	}
	if ev.payload["actor_user_id"] != "user-1" {
		t.Fatalf("actor_user_id = %v, want user-1", ev.payload["actor_user_id"])
	}
	if ev.payload["task_id"] != "task-1" {
		t.Fatalf("task_id = %v, want task-1", ev.payload["task_id"])
	}
}

func TestService_CreateTask_InvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *projectv1.CreateTaskRequest
		wantErr error
	}{
		{
			name: "empty assigned_to",
			req: func() *projectv1.CreateTaskRequest {
				r := validCreateTaskRequest()
				r.AssignedTo = ""
				return r
			}(),
			wantErr: errs.ErrInvalidInput,
		},
		{
			name: "empty title",
			req: func() *projectv1.CreateTaskRequest {
				r := validCreateTaskRequest()
				r.Title = ""
				return r
			}(),
			wantErr: errs.ErrInvalidInput,
		},
		{
			name: "empty project_id",
			req: func() *projectv1.CreateTaskRequest {
				r := validCreateTaskRequest()
				r.ProjectId = ""
				return r
			}(),
			wantErr: errs.ErrInvalidInput,
		},
		{
			name: "unknown activity type",
			req: func() *projectv1.CreateTaskRequest {
				r := validCreateTaskRequest()
				r.ActivityTypeId = "missing-type"
				return r
			}(),
			wantErr: errs.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, ctx, deps := newTestService()
			_, err := svc.CreateTask(ctx, tt.req)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateTask() error = %v, want %v", err, tt.wantErr)
			}
			if len(deps.taskRepo.created) != 0 {
				t.Fatalf("taskRepo.Create called %d times, want 0", len(deps.taskRepo.created))
			}
		})
	}
}

func TestService_CreateTask_PermissionDenied(t *testing.T) {
	t.Parallel()

	svc, ctx, deps := newTestService(func(d *testServiceDeps) {
		d.authz = mockPermissionChecker{allow: false}
	})

	_, err := svc.CreateTask(ctx, validCreateTaskRequest())
	if !errors.Is(err, errs.ErrNoPermission) {
		t.Fatalf("CreateTask() error = %v, want %v", err, errs.ErrNoPermission)
	}
	if len(deps.taskRepo.created) != 0 {
		t.Fatalf("taskRepo.Create called %d times, want 0", len(deps.taskRepo.created))
	}
}

func TestService_CreateTask_PublishFailureStillCreates(t *testing.T) {
	t.Parallel()

	svc, ctx, deps := newTestService(func(d *testServiceDeps) {
		d.publisher.err = errors.New("rabbit down")
	})

	task, err := svc.CreateTask(ctx, validCreateTaskRequest())
	if err != nil {
		t.Fatalf("CreateTask() error = %v, want nil", err)
	}
	if task.Id != "task-1" {
		t.Fatalf("task.Id = %q, want task-1", task.Id)
	}
	if len(deps.taskRepo.created) != 1 {
		t.Fatalf("taskRepo.Create called %d times, want 1", len(deps.taskRepo.created))
	}
}


func TestService_UpdateTaskStatus_NoChange(t *testing.T) {
	t.Parallel()

	svc, ctx, deps := newTestService(withTaskInRepo(&projectv1.Task{
		Id:     "task-1",
		Status: projectv1.TaskStatus_TASK_STATUS_TODO,
	}))

	task, err := svc.UpdateTaskStatus(ctx, &projectv1.UpdateTaskStatusRequest{
		Id:     "task-1",
		Status: projectv1.TaskStatus_TASK_STATUS_TODO,
	})
	if err != nil {
		t.Fatalf("UpdateTaskStatus() error = %v, want nil", err)
	}
	if task.Status != projectv1.TaskStatus_TASK_STATUS_TODO {
		t.Fatalf("Status = %v, want TODO", task.Status)
	}
	if len(deps.publisher.events) != 0 {
		t.Fatalf("published %d events, want 0", len(deps.publisher.events))
	}
}

func TestService_UpdateTaskStatus_Change(t *testing.T) {
	t.Parallel()

	svc, ctx, deps := newTestService(withTaskInRepo(&projectv1.Task{
		Id:        "task-1",
		ProjectId: "project-1",
		Status:    projectv1.TaskStatus_TASK_STATUS_TODO,
	}))

	before, err := deps.taskRepo.FindByID(ctx, "task-1")
	if err != nil {
		t.Fatalf("setup FindByID() error = %v", err)
	}

	task, err := svc.UpdateTaskStatus(ctx, &projectv1.UpdateTaskStatusRequest{
		Id:     "task-1",
		Status: projectv1.TaskStatus_TASK_STATUS_COMPLETED,
	})
	if err != nil {
		t.Fatalf("UpdateTaskStatus() error = %v, want nil", err)
	}
	if before.Status != projectv1.TaskStatus_TASK_STATUS_TODO {
		t.Fatalf("before status = %v, want TODO", before.Status)
	}
	if task.Status != projectv1.TaskStatus_TASK_STATUS_COMPLETED {
		t.Fatalf("Status = %v, want COMPLETED", task.Status)
	}
	if len(deps.publisher.events) != 1 {
		t.Fatalf("published %d events, want 1", len(deps.publisher.events))
	}
	ev := deps.publisher.events[0]
	if ev.routingKey != "task.status_changed" {
		t.Fatalf("routing key = %q, want task.status_changed", ev.routingKey)
	}
	if ev.payload["event_type"] != "task.status_changed" {
		t.Fatalf("event_type = %q, want task.status_changed", ev.payload["event_type"])
	}
	if ev.payload["actor_user_id"] != "user-1" {
		t.Fatalf("actor_user_id = %q, want user-1", ev.payload["actor_user_id"])
	}
	if ev.payload["task_id"] != "task-1" {
		t.Fatalf("task_id = %q, want task-1", ev.payload["task_id"])
	}
	if ev.payload["project_id"] != "project-1" {
		t.Fatalf("project_id = %q, want project-1", ev.payload["project_id"])
	}

	if ev.payload["from_status"] != int32(projectv1.TaskStatus_TASK_STATUS_TODO) {
		t.Fatalf("from_status = %v, want TODO", ev.payload["from_status"])
	}
	if ev.payload["to_status"] != int32(projectv1.TaskStatus_TASK_STATUS_COMPLETED) {
		t.Fatalf("to_status = %v, want COMPLETED", ev.payload["to_status"])
	}
	
}