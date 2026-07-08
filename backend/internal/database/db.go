package database

import (
	"fmt"
	"time"

	"backend/configs"
	"backend/internal/api/models"
	"backend/internal/logger"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(cfg *configs.Config) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)

	var db *gorm.DB
	var err error

	// Retry connection for database startup
	for i := 1; i <= 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		logger.Log.Warn("Failed to connect to database, retrying...",
			zap.Int("attempt", i),
			zap.Error(err),
		)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		logger.Log.Fatal("Could not connect to database after retries", zap.Error(err))
	}

	logger.Log.Info("Successfully connected to the database")

	// Run Auto Migrations
	err = db.AutoMigrate(&models.Sandbox{}, &models.SandboxLog{})
	if err != nil {
		logger.Log.Fatal("Database migration failed", zap.Error(err))
	}

	logger.Log.Info("Database migrations completed successfully")

	// Use Prometheus Telemetry Plugin
	err = db.Use(&TelemetryPlugin{})
	if err != nil {
		logger.Log.Fatal("Failed to register database telemetry plugin", zap.Error(err))
	}

	// Start database stats monitoring
	StartStatsReporting(db)

	DB = db
}
