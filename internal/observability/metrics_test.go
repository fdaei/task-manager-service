package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareRecordsRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	router := gin.New()
	router.Use(m.Middleware())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, 1.0, testutil.ToFloat64(m.requestsTotal.WithLabelValues(http.MethodGet, "/ping", "200")))
}

func TestSetTasksCountUpdatesGauge(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.SetTasksCount(7)

	require.Equal(t, 7.0, testutil.ToFloat64(m.tasksCount))
}

func TestHandlerServesMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	router := gin.New()
	router.GET("/metrics", m.Handler())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Contains(t, resp.Body.String(), "requests_total")
}

func TestRegisterGaugeReusesExisting(t *testing.T) {
	reg := prometheus.NewRegistry()
	initial := prometheus.NewGauge(prometheus.GaugeOpts{Name: "shared"})
	require.NoError(t, reg.Register(initial))

	reused := registerGauge(reg, prometheus.NewGauge(prometheus.GaugeOpts{Name: "shared"}))

	require.Equal(t, initial, reused)
}

func TestRegisterCounterVecReusesExisting(t *testing.T) {
	reg := prometheus.NewRegistry()
	base := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "requests_total"}, []string{"method", "path", "status"})
	require.NoError(t, reg.Register(base))

	reused := registerCounterVec(reg, prometheus.NewCounterVec(prometheus.CounterOpts{Name: "requests_total"}, []string{"method", "path", "status"}))

	require.Equal(t, base, reused)
}

func TestRegisterHistogramReusesExisting(t *testing.T) {
	reg := prometheus.NewRegistry()
	base := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "request_latency_seconds"}, []string{"method", "path", "status"})
	require.NoError(t, reg.Register(base))

	reused := registerHistogramVec(reg, prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "request_latency_seconds"}, []string{"method", "path", "status"}))

	require.Equal(t, base, reused)
}
