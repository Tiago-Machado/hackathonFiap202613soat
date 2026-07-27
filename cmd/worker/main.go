package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"video-processor/internal/adapters/ffmpeg"
	"video-processor/internal/adapters/postgres"
	"video-processor/internal/adapters/rabbitmq"
	"video-processor/internal/adapters/smtp"
	"video-processor/internal/config"
	"video-processor/internal/observability"
	"video-processor/internal/retention"
	"video-processor/internal/storage"
	"video-processor/internal/usecase"
)

const hoursPerDay = 24

func main() {
	log := observability.NewLogger()

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
	userRepo := postgres.NewUserRepository(db)
	extractor := ffmpeg.NewExtractor(store)
	notifier := smtp.NewNotifier(cfg.SMTPHost, cfg.SMTPPort)
	retentionWindow := time.Duration(cfg.RetentionDays) * hoursPerDay * time.Hour

	processor := usecase.NewProcessVideo(videoRepo, extractor, notifier, userRepo, retentionWindow, clock)

	consumer, err := rabbitmq.NewConsumer(broker, processor, log)
	if err != nil {
		log.Error("consumer_indisponivel", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	purger := retention.New(db, store, log)
	go purger.Run(ctx)

	go func() {
		log.Info("worker_iniciado")
		if err := consumer.Run(ctx); err != nil && ctx.Err() == nil {
			log.Error("consumer_encerrado_com_erro", "error", err)
		}
	}()

	waitForShutdown()
	cancel()
	log.Info("worker_encerrado")
}

func waitForShutdown() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
}
