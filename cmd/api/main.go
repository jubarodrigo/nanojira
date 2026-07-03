package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rodrigocavalhero/nanojira/internal/config"
	"github.com/rodrigocavalhero/nanojira/internal/email"
	"github.com/rodrigocavalhero/nanojira/internal/handler"
	"github.com/rodrigocavalhero/nanojira/internal/handler/middleware"
	"github.com/rodrigocavalhero/nanojira/internal/logger"
	"github.com/rodrigocavalhero/nanojira/internal/repository/postgres"
	"github.com/rodrigocavalhero/nanojira/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	ctx := context.Background()

	if err := postgres.RunMigrations(cfg.DatabaseURL, cfg.MigrationsDir); err != nil {
		log.Fatal("run migrations", zap.Error(err))
	}

	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("connect database", zap.Error(err))
	}
	defer db.Close()

	userRepo := postgres.NewUserRepo(db)
	taskRepo := postgres.NewTaskRepo(db)
	stepBackRepo := postgres.NewStepBackRepo(db)
	emailSender := email.NewSMTPSender(email.SMTPConfig{
		Host: cfg.SMTPHost,
		Port: cfg.SMTPPort,
		From: cfg.SMTPFrom,
	})

	svc := service.New(userRepo, taskRepo, stepBackRepo, emailSender, log)
	h := handler.New(svc)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(log))
	h.Register(router)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		log.Info("server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown", zap.Error(err))
	}
}
