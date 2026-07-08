package database

import (
	"time"

	"backend/internal/metrics"

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

		metrics.DbQueryDuration.WithLabelValues(queryType, tableName).Observe(duration)

		if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
			metrics.DbQueryErrorsTotal.WithLabelValues(queryType, tableName).Inc()
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
