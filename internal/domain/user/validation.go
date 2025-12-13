package user

import (
	"context"
	"errors"
)

type Validator interface {
	ValidateCreate(ctx context.Context, params CreateUserParams) error
}

type OzzoValidator struct {
	userRepo Repository
}

func NewValidator(userRepo Repository) Validator {
	return &OzzoValidator{userRepo: userRepo}
}

func (v *OzzoValidator) ValidateCreate(ctx context.Context, params CreateUserParams) error {
	if err := params.Validate(); err != nil {
		return err
	}

	return v.ensureEmailUnique(ctx, params.Email)
}

func (v *OzzoValidator) ensureEmailUnique(ctx context.Context, email string) error {
	if v.userRepo == nil {
		return nil
	}

	exists, err := v.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return err
	}

	if exists {
		return errors.New("email already exists")
	}

	return nil
}
