package http

import (
	"context"
	"errors"
	nethttp "net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"taskservice/internal/domain/task"
	"taskservice/internal/domain/user"
	"taskservice/internal/observability"
	"taskservice/pkg/httperr"
	"taskservice/pkg/types"
)

type TaskService interface {
	CreateTask(ctx context.Context, params task.CreateTaskParams) (*task.Task, error)
	GetTask(ctx context.Context, id types.ID) (*task.Task, error)
	ListTasks(ctx context.Context, params task.ListTasksParams) ([]task.Task, error)
	UpdateTask(ctx context.Context, params task.UpdateTaskParams) (*task.Task, error)
	DeleteTask(ctx context.Context, id types.ID) error
	CountTasks(ctx context.Context, params task.ListTasksParams) (int, error)
}

type UserService interface {
	CreateUser(ctx context.Context, params user.CreateUserParams) (*user.User, error)
	GetUser(ctx context.Context, id types.ID) (*user.User, error)
}

type TaskHandler struct {
	taskService TaskService
	userService UserService
	metrics     *observability.Metrics
}

func NewHandler(taskService TaskService, userService UserService, metrics *observability.Metrics) *TaskHandler {
	if metrics == nil {
		metrics = observability.NewMetrics(nil)
	}

	return &TaskHandler{
		taskService: taskService,
		userService: userService,
		metrics:     metrics,
	}
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req task.CreateTaskParams
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, nethttp.StatusBadRequest, err)
		return
	}

	params := task.CreateTaskParams{
		UserID:      req.UserID,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	}

	createdTask, err := h.taskService.CreateTask(c.Request.Context(), params)
	if err != nil {
		writeError(c, httperr.MapServiceError(err), err)
		return
	}

	h.refreshTasksCount(c.Request.Context())
	c.JSON(nethttp.StatusCreated, toTaskResponse(*createdTask))
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		writeError(c, nethttp.StatusBadRequest, err)
		return
	}

	t, err := h.taskService.GetTask(c.Request.Context(), id)
	if err != nil {
		writeError(c, httperr.MapServiceError(err), err)
		return
	}

	c.JSON(nethttp.StatusOK, toTaskResponse(*t))
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
	listParams, err := parseListParams(c)
	if err != nil {
		writeError(c, nethttp.StatusBadRequest, err)
		return
	}

	tasks, err := h.taskService.ListTasks(c.Request.Context(), listParams)
	if err != nil {
		writeError(c, httperr.MapServiceError(err), err)
		return
	}

	responses := make([]task.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		responses = append(responses, toTaskResponse(t))
	}

	if listParams.UserID == nil && listParams.Status == nil {
		if total, err := h.taskService.CountTasks(c.Request.Context(), listParams); err == nil {
			h.setTasksCount(total)
		}
	}

	c.JSON(nethttp.StatusOK, responses)
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		writeError(c, nethttp.StatusBadRequest, err)
		return
	}

	var req task.UpdateTaskParams
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, nethttp.StatusBadRequest, err)
		return
	}

	params := task.UpdateTaskParams{
		ID:          id,
		UserID:      req.UserID,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	}

	updatedTask, err := h.taskService.UpdateTask(c.Request.Context(), params)
	if err != nil {
		writeError(c, httperr.MapServiceError(err), err)
		return
	}

	c.JSON(nethttp.StatusOK, toTaskResponse(*updatedTask))
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		writeError(c, nethttp.StatusBadRequest, err)
		return
	}

	if err := h.taskService.DeleteTask(c.Request.Context(), id); err != nil {
		writeError(c, httperr.MapServiceError(err), err)
		return
	}

	h.refreshTasksCount(c.Request.Context())
	c.Status(nethttp.StatusNoContent)
}

func (h *TaskHandler) CreateUser(c *gin.Context) {
	var req user.CreateUserParams
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, nethttp.StatusBadRequest, err)
		return
	}

	if err := req.Validate(); err != nil {
		writeError(c, nethttp.StatusBadRequest, err)
		return
	}

	createdUser, err := h.userService.CreateUser(c.Request.Context(), req)
	if err != nil {
		writeError(c, httperr.MapServiceError(err), err)
		return
	}

	c.JSON(nethttp.StatusCreated, toUserResponse(*createdUser))
}

func (h *TaskHandler) GetUser(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		writeError(c, nethttp.StatusBadRequest, err)
		return
	}

	u, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		writeError(c, httperr.MapServiceError(err), err)
		return
	}

	c.JSON(nethttp.StatusOK, toUserResponse(*u))
}

func toTaskResponse(t task.Task) task.TaskResponse {
	return task.TaskResponse{
		ID:          t.ID,
		UserID:      t.UserID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func toUserResponse(u user.User) user.UserResponse {
	return user.UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func parseIDParam(raw string) (types.ID, error) {
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return types.ID(parsed), nil
}

func parseOptionalID(raw string) (*types.ID, error) {
	if raw == "" {
		return nil, nil
	}

	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return nil, err
	}

	parsed := types.ID(value)
	return &parsed, nil
}

func parseListParams(c *gin.Context) (task.ListTasksParams, error) {
	var params task.ListTasksParams

	if userID, err := parseOptionalID(c.Query("user_id")); err != nil {
		return params, err
	} else {
		params.UserID = userID
	}

	if status := c.Query("status"); status != "" {
		if !isValidStatus(status) {
			return params, errors.New("invalid status")
		}
		params.Status = &status
	}

	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	params.Limit = pageSize
	params.Offset = (page - 1) * pageSize

	return params, nil
}

func isValidStatus(status string) bool {
	switch status {
	case "todo", "doing", "done":
		return true
	default:
		return false
	}
}

func parsePositiveInt(raw string, def int) int {
	if raw == "" {
		return def
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return def
	}
	return value
}

func writeError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"error": err.Error()})
}

func (h *TaskHandler) refreshTasksCount(ctx context.Context) {
	if h.metrics == nil {
		return
	}

	if total, err := h.taskService.CountTasks(ctx, task.ListTasksParams{}); err == nil {
		h.metrics.SetTasksCount(total)
	}
}

func (h *TaskHandler) setTasksCount(count int) {
	if h.metrics == nil {
		return
	}

	h.metrics.SetTasksCount(count)
}
