package authz

import (
	projectv1 "building-services/gen/project/v1"
	"building-services/project-service/internal/user"
	"context"
	"database/sql"
	"testing"
)

const testProjectID = "project-1"
const testDeptID = "dept-1"

func strPtr(s string) *string { return &s }
func memberKey(a, b string) string { return a + ":" + b }

type mockUserRepo struct {
	users map[string]*user.User
}

func (m *mockUserRepo) FindByID(_ context.Context, id string) (*user.User, error) {
	return m.users[id], nil
}

type mockMemberRepo struct {
	members  map[string]bool
	inDept   map[string]bool
	managers map[string]bool
}

func (m mockMemberRepo) IsProjectMember(_ context.Context, projectID, userID string) (*projectv1.ProjectMember, error) {
	if m.members != nil && m.members[memberKey(projectID, userID)] {
		return &projectv1.ProjectMember{ProjectId: projectID, UserId: userID}, nil
	}
	return nil, sql.ErrNoRows
}

func (m mockMemberRepo) IsProjectInDepartment(_ context.Context, projectID, departmentID string) (bool, error) {
	if m.inDept == nil {
		return false, nil
	}
	return m.inDept[memberKey(projectID, departmentID)], nil
}

func (m mockMemberRepo) IsManagerOfProject(_ context.Context, userID, projectID string) (bool, error) {
	if m.managers == nil {
		return false, nil
	}
	return m.managers[memberKey(userID, projectID)], nil
}

func (m mockMemberRepo) GetProjectMembers(context.Context, string) ([]*projectv1.ProjectMember, error) {
	return nil, nil
}

type mockTaskRepo struct {
	projectIDs map[string]string
	assignees  map[string]bool
	tasks      map[string]*projectv1.Task
}

func (m mockTaskRepo) FindByID(_ context.Context, id string) (*projectv1.Task, error) {
	if m.tasks != nil {
		if task, ok := m.tasks[id]; ok {
			return task, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m mockTaskRepo) GetProjectID(_ context.Context, id string) (string, error) {
	if m.projectIDs != nil {
		if projectID, ok := m.projectIDs[id]; ok {
			return projectID, nil
		}
	}
	return "", sql.ErrNoRows
}

func (m mockTaskRepo) IsAssignee(_ context.Context, taskID, userID string) (bool, error) {
	if m.assignees == nil {
		return false, nil
	}
	return m.assignees[memberKey(taskID, userID)], nil
}

type mockAttachmentRepo struct {
	taskIDs     map[string]string
	uploadedBy  map[string]string
}

func (m mockAttachmentRepo) GetTaskID(_ context.Context, attachmentID string) (string, error) {
	if m.taskIDs != nil {
		if taskID, ok := m.taskIDs[attachmentID]; ok {
			return taskID, nil
		}
	}
	return "", sql.ErrNoRows
}

func (m mockAttachmentRepo) GetUploadedBy(_ context.Context, attachmentID string) (string, error) {
	if m.uploadedBy != nil {
		return m.uploadedBy[attachmentID], nil
	}
	return "", nil
}

type mockDepartmentRepo struct {
	departments map[string]*projectv1.Department
}

func (m mockDepartmentRepo) FindByID(_ context.Context, id string) (*projectv1.Department, error) {
	if m.departments != nil {
		if dept, ok := m.departments[id]; ok {
			return dept, nil
		}
	}
	return nil, sql.ErrNoRows
}

type testDeps struct {
	users       map[string]*user.User
	members     mockMemberRepo
	tasks       mockTaskRepo
	attachments mockAttachmentRepo
	departments mockDepartmentRepo
}

func newTestChecker(d testDeps) *PermissionChecker {
	if d.users == nil {
		d.users = testUsers()
	}
	return NewPermissionChecker(
		&mockUserRepo{users: d.users},
		d.members,
		d.tasks,
		d.attachments,
		d.departments,
	)
}

func testUsers() map[string]*user.User {
	return map[string]*user.User{
		"pm-1":                 {ID: "pm-1", Role: RoleProjectManager},
		"worker-1":             {ID: "worker-1", Role: RoleWorker},
		"worker-2":             {ID: "worker-2", Role: RoleWorker},
		"worker-3":             {ID: "worker-3", Role: RoleWorker},
		"director-1":           {ID: "director-1", Role: RoleDirector},
		"gip-1":                {ID: "gip-1", Role: RoleGIP},
		"department-manager-1": {ID: "department-manager-1", Role: RoleDepartmentManager, DepartmentID: strPtr(testDeptID)},
		"department-manager-2": {ID: "department-manager-2", Role: RoleDepartmentManager, DepartmentID: strPtr("dept-other")},
		"admin-1":              {ID: "admin-1", Role: RoleAdmin},
	}
}

func TestPermissionChecker_Check_CreateProject(t *testing.T) {
	t.Parallel()

	checker := newTestChecker(testDeps{})

	tests := []struct {
		name    string
		wantOK  bool
		wantErr bool
	}{
		{name: "pm-1", wantOK: true},
		{name: "gip-1", wantOK: true},
		{name: "director-1", wantOK: true},
		{name: "admin-1", wantOK: true},
		{name: "worker-1", wantOK: false},
		{name: "department-manager-1", wantOK: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotOK, err := checker.Check(
				context.Background(),
				tt.name,
				ResourceProject,
				"",
				ActionCreate,
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotOK != tt.wantOK {
				t.Fatalf("Check() = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestPermissionChecker_Check_ViewProject(t *testing.T) {
	t.Parallel()

	checker := newTestChecker(testDeps{
		members: mockMemberRepo{
			members: map[string]bool{
				memberKey(testProjectID, "pm-1"):  true,
				memberKey(testProjectID, "gip-1"): true,
			},
			inDept: map[string]bool{
				memberKey(testProjectID, testDeptID): true,
			},
		},
	})

	tests := []struct {
		name    string
		wantOK  bool
		wantErr bool
	}{
		{name: "pm-1", wantOK: true},
		{name: "gip-1", wantOK: true},
		{name: "worker-1", wantOK: false},
		{name: "director-1", wantOK: true},
		{name: "admin-1", wantOK: true},
		{name: "department-manager-1", wantOK: true},
		{name: "department-manager-2", wantOK: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotOK, err := checker.Check(
				context.Background(),
				tt.name,
				ResourceProject,
				testProjectID,
				ActionView,
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotOK != tt.wantOK {
				t.Fatalf("Check() = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestPermissionChecker_Check_EditProject(t *testing.T) {
    t.Parallel()

	checker := newTestChecker(testDeps{
		members: mockMemberRepo{
			members:map[string]bool{
				memberKey(testProjectID, "gip-1"): true,
		},
		managers: map[string]bool{
			memberKey("pm-1", testProjectID): true,
		},
		inDept: map[string]bool{
			memberKey(testProjectID, testDeptID): true,
		},
	},
	})
	
    tests := []struct {
		name    string
		wantOK  bool
		wantErr bool
	}{
		{name: "gip-1", wantOK: true},
		{name: "pm-1", wantOK: true},
		{name: "worker-1", wantOK: false},
		{name: "director-1", wantOK: true},
		{name: "admin-1", wantOK: true},
		{name: "department-manager-1", wantOK: true},
		{name: "department-manager-2", wantOK: false},
	}

	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotOK, err := checker.Check(
				context.Background(),
				tt.name,
				ResourceProject,
				testProjectID,
				ActionEdit,
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotOK != tt.wantOK {
				t.Fatalf("Check() = %v, want %v", gotOK, tt.wantOK)
			}
	})
}

}

const testTaskID = "task-1"

func taskTestDeps(opts ...func(*testDeps)) testDeps {
	d := testDeps{
		tasks: mockTaskRepo{
			projectIDs: map[string]string{testTaskID: testProjectID},
			assignees: map[string]bool{
				memberKey(testTaskID, "worker-1"): true,
			},
			tasks: map[string]*projectv1.Task{
				testTaskID: {Id: testTaskID, CreatedBy: "worker-2"},
			},
		},
	}
	for _, opt := range opts {
		opt(&d)
	}
	return d
}

func TestPermissionChecker_Check_ViewTask(t *testing.T) {
	t.Parallel()
	checker := newTestChecker(taskTestDeps())
	tests := []struct {
		name    string
		wantOK  bool
		wantErr bool
	}{
		{name: "worker-1", wantOK: true},
		{name: "worker-2", wantOK: false},
		{name: "worker-3", wantOK: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotOK, err := checker.Check(
				context.Background(),
				tt.name,
				ResourceTask,
				testTaskID,
				ActionView,
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotOK != tt.wantOK {
				t.Fatalf("Check() = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestPermissionChecker_Check_EditTask(t *testing.T) {
	t.Parallel()
	checker := newTestChecker(taskTestDeps())
	tests := []struct {
		name    string
		wantOK  bool
		wantErr bool
	}{
		{name: "worker-1", wantOK: false},
		{name: "worker-2", wantOK: false},
		{name: "worker-3", wantOK: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotOK, err := checker.Check(
				context.Background(),
				tt.name,
				ResourceTask,
				testTaskID,
				ActionEdit,
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotOK != tt.wantOK {
				t.Fatalf("Check() = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestPermissionChecker_Check_DeleteTask(t *testing.T) {
	t.Parallel()
	checker := newTestChecker(taskTestDeps())
	tests := []struct {
		name    string
		wantOK  bool
		wantErr bool
	}{
		{name: "worker-1", wantOK: false},
		{name: "worker-2", wantOK: true},
		{name: "worker-3", wantOK: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotOK, err := checker.Check(
				context.Background(),
				tt.name,
				ResourceTask,
				testTaskID,
				ActionDelete,
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotOK != tt.wantOK {
				t.Fatalf("Check() = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}