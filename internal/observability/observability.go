package observability

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const unknownRoute = "unknown"

func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

var (
	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total de requisições HTTP por método, rota e status.",
		},
		[]string{"method", "path", "status"},
	)

	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duração das requisições HTTP.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	VideosProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "videos_processed_total",
			Help: "Total de vídeos processados por status final.",
		},
		[]string{"status"},
	)
)

func MustRegister() {
	prometheus.MustRegister(httpRequests, httpDuration, VideosProcessed)
}

func MetricsHandler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

func Middleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		path := c.FullPath()
		if path == "" {
			path = unknownRoute
		}
		status := strconv.Itoa(c.Writer.Status())

		httpRequests.WithLabelValues(c.Request.Method, path, status).Inc()
		httpDuration.WithLabelValues(c.Request.Method, path).Observe(duration.Seconds())

		log.Info("http_request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"ip", c.ClientIP(),
		)
	}
}
