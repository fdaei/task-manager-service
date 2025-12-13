package task

import (
	"time"

	"taskservice/pkg/types"
)

type CreateTaskParams struct {
	UserID      types.ID `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type UpdateTaskParams struct {
	ID          types.ID `json:"id"`
	UserID      types.ID `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type TaskRequest struct {
	UserID      types.ID `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type TaskResponse struct {
	ID          types.ID  `json:"id"`
	UserID      types.ID  `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListTasksParams struct {
	UserID *types.ID
	Status *string
	Limit  int
	Offset int
}
