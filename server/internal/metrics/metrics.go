package metrics

import (
	"database/sql"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_request_total",
			Help: "Total number of API requests.",
		},
		[]string{"method", "path", "status"},
	)

	errorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_error_total",
			Help: "Total number of API error responses.",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_request_duration_seconds",
			Help:    "API request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	dbCollector     = newDBStatsCollector()
	registerDBStats sync.Once
)

func init() {
	prometheus.MustRegister(requestTotal, errorTotal, requestDuration)
}

type dbStatsCollector struct {
	db                  atomic.Pointer[sql.DB]
	maxOpenConnections  *prometheus.Desc
	openConnections     *prometheus.Desc
	inUseConnections    *prometheus.Desc
	idleConnections     *prometheus.Desc
	waitCount           *prometheus.Desc
	waitDurationSeconds *prometheus.Desc
}

func newDBStatsCollector() *dbStatsCollector {
	return &dbStatsCollector{
		maxOpenConnections: prometheus.NewDesc(
			"gva_db_max_open_connections",
			"Maximum number of open database connections.",
			nil,
			nil,
		),
		openConnections: prometheus.NewDesc(
			"gva_db_open_connections",
			"Number of established database connections, both in use and idle.",
			nil,
			nil,
		),
		inUseConnections: prometheus.NewDesc(
			"gva_db_in_use",
			"Number of database connections currently in use.",
			nil,
			nil,
		),
		idleConnections: prometheus.NewDesc(
			"gva_db_idle",
			"Number of idle database connections.",
			nil,
			nil,
		),
		waitCount: prometheus.NewDesc(
			"gva_db_wait_count_total",
			"Total number of database connections waited for.",
			nil,
			nil,
		),
		waitDurationSeconds: prometheus.NewDesc(
			"gva_db_wait_duration_seconds_total",
			"Total time blocked waiting for a new database connection in seconds.",
			nil,
			nil,
		),
	}
}

func (c *dbStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.maxOpenConnections
	ch <- c.openConnections
	ch <- c.inUseConnections
	ch <- c.idleConnections
	ch <- c.waitCount
	ch <- c.waitDurationSeconds
}

func (c *dbStatsCollector) Collect(ch chan<- prometheus.Metric) {
	db := c.db.Load()
	if db == nil {
		return
	}

	stats := db.Stats()
	ch <- prometheus.MustNewConstMetric(c.maxOpenConnections, prometheus.GaugeValue, float64(stats.MaxOpenConnections))
	ch <- prometheus.MustNewConstMetric(c.openConnections, prometheus.GaugeValue, float64(stats.OpenConnections))
	ch <- prometheus.MustNewConstMetric(c.inUseConnections, prometheus.GaugeValue, float64(stats.InUse))
	ch <- prometheus.MustNewConstMetric(c.idleConnections, prometheus.GaugeValue, float64(stats.Idle))
	ch <- prometheus.MustNewConstMetric(c.waitCount, prometheus.CounterValue, float64(stats.WaitCount))
	ch <- prometheus.MustNewConstMetric(c.waitDurationSeconds, prometheus.CounterValue, stats.WaitDuration.Seconds())
}

// RegisterDBStats registers the database collector once and updates the
// database connection used for subsequent scrapes.
func RegisterDBStats(db *sql.DB) {
	if db == nil {
		return
	}

	dbCollector.db.Store(db)
	registerDBStats.Do(func() {
		prometheus.MustRegister(dbCollector)
	})
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL != nil && c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		method := c.Request.Method
		statusCode := c.Writer.Status()
		status := strconv.Itoa(statusCode)

		requestTotal.WithLabelValues(method, path, status).Inc()
		requestDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())

		if statusCode >= http.StatusBadRequest {
			errorTotal.WithLabelValues(method, path, status).Inc()
		}
	}
}

func Handler() gin.HandlerFunc {
	handler := promhttp.Handler()

	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
