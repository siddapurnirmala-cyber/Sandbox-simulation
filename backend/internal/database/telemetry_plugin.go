package database

import (
	"fmt"
	"time"

	"backend/internal/api/services"
	"backend/internal/logger"
	"backend/internal/metrics"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TelemetryPlugin struct{}

func (p *TelemetryPlugin) Name() string {
	return "prometheus_telemetry"
}

func (p *TelemetryPlugin) Initialize(db *gorm.DB) error {
	// Register before callbacks to capture timestamp
	_ = db.Callback().Create().Before("gorm:create").Register("telemetry:before_create", beforeCallback)
	_ = db.Callback().Query().Before("gorm:query").Register("telemetry:before_query", beforeCallback)
	_ = db.Callback().Update().Before("gorm:update").Register("telemetry:before_update", beforeCallback)
	_ = db.Callback().Delete().Before("gorm:delete").Register("telemetry:before_delete", beforeCallback)

	// Register after callbacks to calculate latency and track query errors
	_ = db.Callback().Create().After("gorm:create").Register("telemetry:after_create", afterCallback("INSERT"))
	_ = db.Callback().Query().After("gorm:query").Register("telemetry:after_query", afterCallback("SELECT"))
	_ = db.Callback().Update().After("gorm:update").Register("telemetry:after_update", afterCallback("UPDATE"))
	_ = db.Callback().Delete().After("gorm:delete").Register("telemetry:after_delete", afterCallback("DELETE"))

	return nil
}

func beforeCallback(db *gorm.DB) {
	db.InstanceSet("query_start_time", time.Now())
	
	// Intercept DB query for simulated failures or delays
	if err := services.FailureConfig.InterceptDBQuery(); err != nil {
		db.Error = err
	}
}

func afterCallback(queryType string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		val, exists := db.InstanceGet("query_start_time")
		if !exists {
			return
		}
		startTime, ok := val.(time.Time)
		if !ok {
			return
		}

		duration := time.Since(startTime).Seconds()
		tableName := db.Statement.Table
		if tableName == "" && db.Statement.Schema != nil {
			tableName = db.Statement.Schema.Table
		}
		if tableName == "" {
			tableName = "unknown"
		}

		// Extract HTTP route path from context if GORM query is executed within a Gin request
		endpoint := "background"
		if db.Statement.Context != nil {
			if ginCtx, ok := db.Statement.Context.(*gin.Context); ok && ginCtx != nil {
				endpoint = ginCtx.FullPath()
				if endpoint == "" {
					endpoint = ginCtx.Request.URL.Path
				}
			}
		}

		metrics.DbQueryDuration.WithLabelValues(queryType, tableName, endpoint).Observe(duration)

		if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
			metrics.DbQueryErrorsTotal.WithLabelValues(queryType, tableName).Inc()
		}

		// Trigger SMTP email alert if DB query duration exceeds 500ms (0.5 seconds) SLA
		if duration >= 0.5 {
			logger.Log.Warn("Database Query SLA breach detected! Sending email alert...",
				zap.String("query_type", queryType),
				zap.String("table", tableName),
				zap.String("endpoint", endpoint),
				zap.Float64("duration_seconds", duration),
			)
			reason := "Database Query SLA Breach (>500ms)"
			action := "Investigate database query performance. Analyze query plan (EXPLAIN ANALYZE), add indexes, or tune postgres configs."
			latencyDuration := time.Duration(duration * float64(time.Second))
			go services.Email.SendLatencyAlert(queryType, fmt.Sprintf("DB Query: %s on %s", tableName, endpoint), "db-query", latencyDuration, "127.0.0.1", reason, action)
		}
	}
}

// StartStatsReporting regularly pulls database pool configurations and pushes to Prometheus
func StartStatsReporting(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := sqlDB.Stats()
			metrics.ActiveDatabaseConnections.Set(float64(stats.InUse))
		}
	}()
}
