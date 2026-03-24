package user

type User struct {
	ID           string
	FullName     string
	Email        string
	Role         string
	DepartmentID *string
}
