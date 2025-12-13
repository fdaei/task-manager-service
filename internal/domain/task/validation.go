package task

import (
	"context"
	"errors"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"taskservice/internal/domain/user"
	"taskservice/pkg/types"
)

type Validator interface {
	ValidateCreate(ctx context.Context, params CreateTaskParams) error
	ValidateUpdate(ctx context.Context, params UpdateTaskParams) error
}

type OzzoValidator struct {
	userRepo user.Repository
}

func NewValidator(userRepo user.Repository) Validator {
	return &OzzoValidator{userRepo: userRepo}
}

func (t Task) Validate() error {
	return validateTaskFields(t.UserID, t.Title, t.Status)
}

func (p CreateTaskParams) Validate() error {
	return validateTaskFields(p.UserID, p.Title, p.Status)
}

func (p UpdateTaskParams) Validate() error {
	if err := validation.ValidateStruct(&p,
		validation.Field(&p.ID, validation.Required, validation.Min(types.ID(1))),
	); err != nil {
		return err
	}

	return validateTaskFields(p.UserID, p.Title, p.Status)
}

func (v *OzzoValidator) ValidateCreate(ctx context.Context, params CreateTaskParams) error {
	if err := params.Validate(); err != nil {
		return err
	}

	return v.ensureUserExists(ctx, params.UserID)
}

func (v *OzzoValidator) ValidateUpdate(ctx context.Context, params UpdateTaskParams) error {
	if err := params.Validate(); err != nil {
		return err
	}

	return v.ensureUserExists(ctx, params.UserID)
}

func (v *OzzoValidator) ensureUserExists(ctx context.Context, userID types.ID) error {
	if v.userRepo == nil {
		return nil
	}

	exists, err := v.userRepo.Exists(ctx, userID)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("user not found")
	}

	return nil
}

func validateTaskFields(userID types.ID, title, status string) error {
	payload := struct {
		UserID types.ID
		Title  string
		Status string
	}{
		UserID: userID,
		Title:  title,
		Status: status,
	}

	return validation.ValidateStruct(&payload,
		validation.Field(&payload.UserID, validation.Required, validation.Min(types.ID(1))),
		validation.Field(&payload.Title, validation.Required),
		validation.Field(&payload.Status, validation.Required, validation.In("todo", "doing", "done")),
	)
}
