package task

import (
	"context"
	"errors"
	"strconv"
	"time"

	"taskservice/pkg/types"
)

type Repository interface {
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id types.ID) (*Task, error)
	List(ctx context.Context, params ListTasksParams) ([]Task, error)
	Count(ctx context.Context, params ListTasksParams) (int, error)
	Update(ctx context.Context, t *Task) error
	Delete(ctx context.Context, id types.ID) error
}

type Service struct {
	repository Repository
	validator  Validator
	cache      Cache
}

func NewService(repository Repository, validator Validator, cache Cache) *Service {
	return &Service{
		repository: repository,
		validator:  validator,
		cache:      cache,
	}
}

func (s *Service) CreateTask(ctx context.Context, params CreateTaskParams) (*Task, error) {
	if s.validator == nil {
		return nil, errors.New("validator is nil")
	}

	if err := s.validator.ValidateCreate(ctx, params); err != nil {
		return nil, err
	}

	task := Task{
		UserID:      params.UserID,
		Title:       params.Title,
		Description: params.Description,
		Status:      params.Status,
	}

	if err := s.repository.Create(ctx, &task); err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.InvalidateAll(ctx)
	}

	return &task, nil
}

func (s *Service) GetTask(ctx context.Context, id types.ID) (*Task, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) ListTasks(ctx context.Context, params ListTasksParams) ([]Task, error) {
	cacheKey := listCacheKey(params)
	if s.cache != nil && cacheKey != "" {
		if cached, err := s.cache.GetTasks(ctx, cacheKey); err == nil && cached != nil {
			return cached, nil
		}
	}

	tasks, err := s.repository.List(ctx, params)
	if err != nil {
		return nil, err
	}

	if s.cache != nil && cacheKey != "" {
		_ = s.cache.SetTasks(ctx, cacheKey, tasks, cacheTTL)
	}

	return tasks, nil
}

func (s *Service) UpdateTask(ctx context.Context, params UpdateTaskParams) (*Task, error) {
	if s.validator == nil {
		return nil, errors.New("validator is nil")
	}

	if err := s.validator.ValidateUpdate(ctx, params); err != nil {
		return nil, err
	}

	task := Task{
		ID:          params.ID,
		UserID:      params.UserID,
		Title:       params.Title,
		Description: params.Description,
		Status:      params.Status,
	}

	if err := s.repository.Update(ctx, &task); err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.InvalidateAll(ctx)
	}

	return &task, nil
}

func (s *Service) DeleteTask(ctx context.Context, id types.ID) error {
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}

	if s.cache != nil {
		_ = s.cache.InvalidateAll(ctx)
	}

	return nil
}

func (s *Service) CountTasks(ctx context.Context, params ListTasksParams) (int, error) {
	return s.repository.Count(ctx, params)
}

const cacheTTL = 30 * time.Second

func listCacheKey(params ListTasksParams) string {
	if params.Offset != 0 {
		return ""
	}
	if params.UserID == nil && params.Status == nil {
		if params.Limit <= 0 {
			return "tasks:all"
		}
		return "tasks:all:limit:" + strconv.Itoa(params.Limit)
	}
	return ""
}
