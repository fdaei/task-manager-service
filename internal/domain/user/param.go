package user

import (
	"time"

	"taskservice/pkg/types"
)

type CreateUserParams struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserResponse struct {
	ID        types.ID  `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (p CreateUserParams) Validate() error {
	user := User{
		Name:  p.Name,
		Email: p.Email,
	}
	return user.Validate()
}
