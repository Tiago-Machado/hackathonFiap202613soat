package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"video-processor/internal/middleware"
	"video-processor/internal/observability"
)

type RouterConfig struct {
	Handlers    *Handlers
	Verifier    middleware.TokenVerifier
	RateLimiter *middleware.RateLimiter
	Logger      *slog.Logger
}

func NewRouter(cfg RouterConfig) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(observability.Middleware(cfg.Logger))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/metrics", observability.MetricsHandler())

	router.POST("/auth/register", cfg.Handlers.Register)
	router.POST("/auth/login", cfg.Handlers.Login)

	authed := router.Group("/")
	authed.Use(middleware.Auth(cfg.Verifier))
	authed.Use(cfg.RateLimiter.PerUser())
	authed.POST("/videos", cfg.Handlers.Upload)
	authed.GET("/videos", cfg.Handlers.List)

	return router
}
