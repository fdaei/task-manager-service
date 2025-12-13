package task

import (
	"time"

	"taskservice/pkg/types"
)

type Task struct {
	ID          types.ID
	UserID      types.ID
	Title       string
	Description string
	Status      string // "todo", "doing", "done"
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
