package user

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"taskservice/pkg/types"
)

type User struct {
	ID        types.ID
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u User) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Name, validation.Required),
		validation.Field(&u.Email, validation.Required, is.Email),
	)
}
