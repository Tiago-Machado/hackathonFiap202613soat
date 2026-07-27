package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"video-processor/internal/adapters/postgres"
	"video-processor/internal/adapters/rabbitmq"
	"video-processor/internal/auth"
	"video-processor/internal/config"
	"video-processor/internal/httpapi"
	"video-processor/internal/middleware"
	"video-processor/internal/observability"
	"video-processor/internal/storage"
	"video-processor/internal/usecase"
)

const (
	serverAddress      = ":8080"
	shutdownTimeout    = 10 * time.Second
	rateLimitPerMinute = 60
)

func main() {
	log := observability.NewLogger()
	observability.MustRegister()

	cfg, err := config.Load()
	if err != nil {
		log.Error("config_invalida", "error", err)
		os.Exit(1)
	}

	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		log.Error("postgres_indisponivel", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisClient, err := newRedis(cfg.RedisURL)
	if err != nil {
		log.Error("redis_indisponivel", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	store, err := storage.New(cfg.S3InternalEndpoint, cfg.S3PublicEndpoint,
		cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3UseSSL)
	if err != nil {
		log.Error("storage_indisponivel", "error", err)
		os.Exit(1)
	}

	broker, err := rabbitmq.Dial(cfg.RabbitMQURL)
	if err != nil {
		log.Error("rabbitmq_indisponivel", "error", err)
		os.Exit(1)
	}
	defer broker.Close()

	clock := usecase.Clock(time.Now)
	videoRepo := postgres.NewVideoRepository(db)
	quotaRepo := postgres.NewQuotaRepository(db)
	userRepo := postgres.NewUserRepository(db)
	publisher := rabbitmq.NewPublisher(broker)

	tokens := auth.NewTokenService(cfg.JWTSecret)
	hasher := auth.NewBcryptHasher()

	handlers := httpapi.NewHandlers(
		usecase.NewRegister(userRepo, hasher, clock),
		usecase.NewLogin(userRepo, hasher, tokens),
		usecase.NewUploadVideo(videoRepo, quotaRepo, store, publisher, clock),
		usecase.NewListVideos(videoRepo, store, cfg.PresignExpiry),
	)

	router := httpapi.NewRouter(httpapi.RouterConfig{
		Handlers:    handlers,
		Verifier:    tokens,
		RateLimiter: middleware.NewRateLimiter(redisClient, rateLimitPerMinute),
		Logger:      log,
	})

	server := &http.Server{Addr: serverAddress, Handler: router}
	go func() {
		log.Info("api_iniciada", "address", serverAddress)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("api_encerrada_com_erro", "error", err)
		}
	}()

	waitForShutdown()

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = server.Shutdown(ctx)
	log.Info("api_encerrada")
}

func newRedis(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opts), nil
}

func waitForShutdown() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
}
