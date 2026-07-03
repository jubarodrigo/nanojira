package domain

type Role string

const (
	RoleManager Role = "manager"
	RoleWorker  Role = "worker"
)

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  Role   `json:"role"`
}

func (u User) IsManager() bool {
	return u.Role == RoleManager
}

func (u User) IsWorker() bool {
	return u.Role == RoleWorker
}
