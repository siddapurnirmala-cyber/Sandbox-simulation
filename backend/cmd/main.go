package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/configs"
	"backend/internal/api/handlers"
	"backend/internal/database"
	"backend/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	cfg := configs.LoadConfig()

	// Initialize Logger
	logger.InitLogger(cfg.LogFile)
	defer func() {
		_ = logger.Log.Sync()
	}()

	logger.Log.Info("Starting Sandbox Observability Platform backend...")

	// Initialize Database
	database.InitDB(cfg)

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Setup fallback recovery middleware
	r.Use(gin.Recovery())

	// Basic routes
	r.GET("/health", handlers.HealthCheck)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Graceful shutdown execution
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Listen and serve error", zap.Error(err))
		}
	}()

	logger.Log.Info("Backend server running", zap.String("port", cfg.Port))

	// Listen for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down backend server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Log.Info("Backend server exited gracefully")
}
