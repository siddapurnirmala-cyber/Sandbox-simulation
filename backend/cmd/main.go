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
	"backend/internal/api/middleware"
	"backend/internal/database"
	"backend/internal/logger"
	"backend/internal/metrics"

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

	// Register Prometheus metrics
	metrics.RegisterMetrics()

	// Initialize Database
	database.InitDB(cfg)

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Initialize Rate Limiter: allows 50 requests/sec, burst up to 100 per IP
	rl := middleware.NewRateLimiter(50.0, 100.0)

	// Wire global middlewares (order: RequestID -> CORS -> Recovery -> Logging -> Prometheus -> RateLimiter -> Timeout)
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.PrometheusMiddleware())
	r.Use(middleware.FailureSimulatorMiddleware())
	r.Use(middleware.RateLimitMiddleware(rl))
	r.Use(middleware.TimeoutMiddleware(15 * time.Second))

	// Basic routes
	r.GET("/health", handlers.HealthCheck)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Sandbox REST APIs
	r.POST("/sandbox", handlers.CreateSandbox)
	r.GET("/sandbox", handlers.ListSandboxes)
	r.GET("/sandbox/:id", handlers.GetSandbox)
	r.DELETE("/sandbox/:id", handlers.DeleteSandbox)
	r.POST("/sandbox/:id/connect", handlers.ConnectSandbox)
	r.POST("/sandbox/:id/disconnect", handlers.DisconnectSandbox)
	r.POST("/sandbox/:id/run-command", handlers.RunCommandSandbox)
	r.GET("/logs", handlers.GetLogs)

	// Failure Simulation APIs
	r.POST("/simulate/api-delay", handlers.SimulateAPIDelay)
	r.POST("/simulate/db-delay", handlers.SimulateDBDelay)
	r.POST("/simulate/db-failure", handlers.SimulateDBFailure)
	r.POST("/simulate/vsi-timeout", handlers.SimulateVSITimeout)
	r.POST("/simulate/high-memory", handlers.SimulateHighMemory)
	r.POST("/simulate/high-cpu", handlers.SimulateHighCPU)
	r.POST("/simulate/random-errors", handlers.SimulateRandomErrors)

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
