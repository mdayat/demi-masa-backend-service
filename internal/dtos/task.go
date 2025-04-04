package dtos

type CreateTaskRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
}

type UpdateTaskRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Checked     *bool  `json:"checked"`
}

type TaskResponse struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Checked     bool   `json:"checked"`
}
