package user

import (
	"context"

	"taskservice/pkg/types"
)

type Service struct {
	repository Repository
	validator  Validator
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
		validator:  NewValidator(repository),
	}
}

func (s *Service) CreateUser(ctx context.Context, params CreateUserParams) (*User, error) {
	if s.validator != nil {
		if err := s.validator.ValidateCreate(ctx, params); err != nil {
			return nil, err
		}
	} else {
		if err := params.Validate(); err != nil {
			return nil, err
		}
	}

	user := User{
		Name:  params.Name,
		Email: params.Email,
	}

	if err := s.repository.Create(ctx, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Service) GetUser(ctx context.Context, id types.ID) (*User, error) {
	u, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &u, nil
}
