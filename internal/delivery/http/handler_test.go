package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"taskservice/internal/domain/task"
	"taskservice/internal/domain/user"
	"taskservice/internal/observability"
	"taskservice/pkg/types"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestListTasksParsesFilters(t *testing.T) {
	taskSvc := &fakeTaskService{
		listResp: []task.Task{{ID: 1, UserID: 2, Title: "title", Status: "todo"}},
		count:    1,
	}
	router := NewRouter(NewHandler(taskSvc, &fakeUserService{}, nil), false)

	req := httptest.NewRequest(http.MethodGet, "/tasks?user_id=2&status=todo&page=2&page_size=5", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.NotNil(t, taskSvc.listParams.UserID)
	require.Equal(t, types.ID(2), *taskSvc.listParams.UserID)
	require.NotNil(t, taskSvc.listParams.Status)
	require.Equal(t, "todo", *taskSvc.listParams.Status)
	require.Equal(t, 5, taskSvc.listParams.Limit)
	require.Equal(t, 5, taskSvc.listParams.Offset)
}

func TestListTasksRejectsInvalidStatus(t *testing.T) {
	taskSvc := &fakeTaskService{}
	router := NewRouter(NewHandler(taskSvc, &fakeUserService{}, nil), false)

	req := httptest.NewRequest(http.MethodGet, "/tasks?status=bogus", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestCreateTaskSuccess(t *testing.T) {
	taskSvc := &fakeTaskService{
		createResp: &task.Task{ID: 10, UserID: 1, Title: "title", Status: "todo"},
		count:      1,
	}
	router := NewRouter(NewHandler(taskSvc, &fakeUserService{}, nil), false)

	body := bytes.NewBufferString(`{"user_id":1,"title":"title","status":"todo"}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusCreated, resp.Code)
	require.Equal(t, types.ID(1), taskSvc.createParams.UserID)
}

func TestCreateTaskRejectsBadJSON(t *testing.T) {
	taskSvc := &fakeTaskService{}
	router := NewRouter(NewHandler(taskSvc, &fakeUserService{}, nil), false)

	body := bytes.NewBufferString(`{oops`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestCreateUserSuccess(t *testing.T) {
	userSvc := &fakeUserService{
		createResp: &user.User{ID: 5, Name: "demo", Email: "demo@example.com"},
	}
	router := NewRouter(NewHandler(&fakeTaskService{}, userSvc, nil), false)

	body := bytes.NewBufferString(`{"name":"demo","email":"demo@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/users", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusCreated, resp.Code)
}

func TestCreateUserValidationError(t *testing.T) {
	router := NewRouter(NewHandler(&fakeTaskService{}, &fakeUserService{}, nil), false)

	body := bytes.NewBufferString(`{"email":"missing-name@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/users", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestDeleteTask(t *testing.T) {
	taskSvc := &fakeTaskService{}
	router := NewRouter(NewHandler(taskSvc, &fakeUserService{}, nil), false)

	req := httptest.NewRequest(http.MethodDelete, "/tasks/3", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNoContent, resp.Code)
	require.Equal(t, types.ID(3), taskSvc.deleteID)
}

func TestDeleteTaskNotFound(t *testing.T) {
	taskSvc := &fakeTaskService{deleteErr: pgx.ErrNoRows}
	router := NewRouter(NewHandler(taskSvc, &fakeUserService{}, nil), false)

	req := httptest.NewRequest(http.MethodDelete, "/tasks/3", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestListTasksUpdatesMetricsWhenUnfiltered(t *testing.T) {
	reg := prometheus.NewRegistry()
	taskSvc := &fakeTaskService{
		listResp: []task.Task{{ID: 1, UserID: 1, Title: "t", Status: "todo"}},
		count:    3,
	}
	handler := NewHandler(taskSvc, &fakeUserService{}, observability.NewMetrics(reg))
	router := NewRouter(handler, false)

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 3.0, readGauge(t, reg, "tasks_count"))
}

func TestGetUser(t *testing.T) {
	userSvc := &fakeUserService{
		getResp: &user.User{ID: 7, Name: "user", Email: "user@example.com"},
	}
	router := NewRouter(NewHandler(&fakeTaskService{}, userSvc, nil), false)

	req := httptest.NewRequest(http.MethodGet, "/users/7", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestGetTask(t *testing.T) {
	taskSvc := &fakeTaskService{listResp: []task.Task{{ID: 9, UserID: 2, Title: "existing", Status: "doing"}}}
	router := NewRouter(NewHandler(taskSvc, &fakeUserService{}, nil), false)

	req := httptest.NewRequest(http.MethodGet, "/tasks/9", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, types.ID(9), taskSvc.getID)

	var body task.TaskResponse
	decodeResponse(t, resp.Body, &body)
	require.Equal(t, "existing", body.Title)
	require.Equal(t, "doing", body.Status)
}

func TestGetTaskNotFound(t *testing.T) {
	taskSvc := &fakeTaskService{getErr: pgx.ErrNoRows}
	router := NewRouter(NewHandler(taskSvc, &fakeUserService{}, nil), false)

	req := httptest.NewRequest(http.MethodGet, "/tasks/11", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestUpdateTask(t *testing.T) {
	taskSvc := &fakeTaskService{
		updateResp: &task.Task{ID: 4, UserID: 5, Title: "updated", Status: "done"},
	}
	router := NewRouter(NewHandler(taskSvc, &fakeUserService{}, nil), false)

	body := bytes.NewBufferString(`{"user_id":5,"title":"updated","status":"done"}`)
	req := httptest.NewRequest(http.MethodPut, "/tasks/4", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, types.ID(4), taskSvc.updateParams.ID)
	require.Equal(t, types.ID(5), taskSvc.updateParams.UserID)

	var decoded task.TaskResponse
	decodeResponse(t, resp.Body, &decoded)
	require.Equal(t, "updated", decoded.Title)
	require.Equal(t, "done", decoded.Status)
}

func TestUpdateTaskInvalidID(t *testing.T) {
	router := NewRouter(NewHandler(&fakeTaskService{}, &fakeUserService{}, nil), false)

	body := bytes.NewBufferString(`{"user_id":1,"title":"x"}`)
	req := httptest.NewRequest(http.MethodPut, "/tasks/not-a-number", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

// --- fakes ---

type fakeTaskService struct {
	createParams task.CreateTaskParams
	updateParams task.UpdateTaskParams
	listParams   task.ListTasksParams
	deleteID     types.ID
	getID        types.ID
	listResp     []task.Task
	createResp   *task.Task
	updateResp   *task.Task
	count        int
	listErr      error
	createErr    error
	updateErr    error
	deleteErr    error
	getErr       error
}

func (f *fakeTaskService) CreateTask(_ context.Context, params task.CreateTaskParams) (*task.Task, error) {
	f.createParams = params
	return f.createResp, f.createErr
}

func (f *fakeTaskService) GetTask(_ context.Context, id types.ID) (*task.Task, error) {
	f.getID = id
	if f.getErr != nil {
		return nil, f.getErr
	}
	for i := range f.listResp {
		if f.listResp[i].ID == id {
			return &f.listResp[i], nil
		}
	}
	return nil, nil
}

func (f *fakeTaskService) ListTasks(_ context.Context, params task.ListTasksParams) ([]task.Task, error) {
	f.listParams = params
	return f.listResp, f.listErr
}

func (f *fakeTaskService) UpdateTask(_ context.Context, params task.UpdateTaskParams) (*task.Task, error) {
	f.updateParams = params
	if f.updateResp != nil {
		resp := *f.updateResp
		return &resp, f.updateErr
	}
	return nil, f.updateErr
}

func (f *fakeTaskService) DeleteTask(_ context.Context, id types.ID) error {
	f.deleteID = id
	return f.deleteErr
}

func (f *fakeTaskService) CountTasks(_ context.Context, _ task.ListTasksParams) (int, error) {
	return f.count, nil
}

type fakeUserService struct {
	createResp *user.User
	getResp    *user.User
	createErr  error
	getErr     error
}

func (f *fakeUserService) CreateUser(_ context.Context, params user.CreateUserParams) (*user.User, error) {
	if f.createResp != nil {
		resp := *f.createResp
		return &resp, f.createErr
	}
	return nil, f.createErr
}

func (f *fakeUserService) GetUser(_ context.Context, _ types.ID) (*user.User, error) {
	return f.getResp, f.getErr
}

// helper to decode JSON in assertions when needed
func decodeResponse(t *testing.T, body *bytes.Buffer, target any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(body.Bytes(), target))
}

func readGauge(t *testing.T, reg *prometheus.Registry, name string) float64 {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() == name && len(mf.Metric) > 0 && mf.Metric[0].GetGauge() != nil {
			return mf.Metric[0].GetGauge().GetValue()
		}
	}
	return 0
}
