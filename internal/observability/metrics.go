package observability

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)


type Metrics struct {
	requestsTotal     *prometheus.CounterVec
	requestLatency    *prometheus.HistogramVec
	tasksCount        prometheus.Gauge
	metricsHandler    gin.HandlerFunc
	requestMiddleware gin.HandlerFunc
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "requests_total",
				Help: "Total number of HTTP requests processed by the service.",
			},
			[]string{"method", "path", "status"},
		),
		requestLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "request_latency_seconds",
				Help:    "Latency of HTTP requests processed by the service.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		tasksCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tasks_count",
				Help: "Number of tasks currently stored.",
			},
		),
	}

	m.requestsTotal = registerCounterVec(reg, m.requestsTotal)
	m.requestLatency = registerHistogramVec(reg, m.requestLatency)
	m.tasksCount = registerGauge(reg, m.tasksCount)
	m.metricsHandler = gin.WrapH(promhttp.Handler())
	m.requestMiddleware = m.buildMiddleware()

	return m
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return m.requestMiddleware
}


func (m *Metrics) Handler() gin.HandlerFunc {
	return m.metricsHandler
}


func (m *Metrics) SetTasksCount(count int) {
	m.tasksCount.Set(float64(count))
}

func (m *Metrics) buildMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		status := strconv.Itoa(c.Writer.Status())

		m.requestsTotal.WithLabelValues(method, path, status).Inc()
		m.requestLatency.WithLabelValues(method, path, status).Observe(time.Since(start).Seconds())
	}
}

func registerCounterVec(reg prometheus.Registerer, collector *prometheus.CounterVec) *prometheus.CounterVec {
	if err := reg.Register(collector); err != nil {
		if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if vec, ok := existing.ExistingCollector.(*prometheus.CounterVec); ok {
				return vec
			}
		} else {
			panic(err)
		}
	}

	return collector
}

func registerHistogramVec(reg prometheus.Registerer, collector *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := reg.Register(collector); err != nil {
		if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if vec, ok := existing.ExistingCollector.(*prometheus.HistogramVec); ok {
				return vec
			}
		} else {
			panic(err)
		}
	}

	return collector
}

func registerGauge(reg prometheus.Registerer, collector prometheus.Gauge) prometheus.Gauge {
	if err := reg.Register(collector); err != nil {
		if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if gauge, ok := existing.ExistingCollector.(prometheus.Gauge); ok {
				return gauge
			}
		} else {
			panic(err)
		}
	}

	return collector
}
