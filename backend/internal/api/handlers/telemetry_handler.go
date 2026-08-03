package handlers

import (
	"net/http"
	"time"

	"backend/internal/api/services"
	"backend/internal/logger"
	"backend/internal/metrics"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UILoadInput struct {
	LoadTimeMs float64 `json:"load_time_ms"`
}

// POST /telemetry/ui-load
func ReportUILoadTime(c *gin.Context) {
	var input UILoadInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Read simulated UI delay if configured
	services.FailureConfig.Lock()
	uiDelayMs := services.FailureConfig.UIDelay.Milliseconds()
	services.FailureConfig.Unlock()

	effectiveLoadTimeMs := input.LoadTimeMs
	if uiDelayMs > 0 {
		effectiveLoadTimeMs += float64(uiDelayMs)
	}

	// Record in Prometheus (convert ms to seconds)
	metrics.UiLoadDuration.Observe(effectiveLoadTimeMs / 1000.0)

	// Trigger SMTP email alert if UI load time exceeds 500ms SLA
	if effectiveLoadTimeMs >= 500.0 {
		logger.Log.Warn("UI SLA breach detected! Sending email alert...",
			zap.Float64("load_time_ms", effectiveLoadTimeMs),
		)
		reason := "React UI Page Load SLA Breach (>500ms)"
		action := "Investigate ROKS performance. Check frontend asset loading speeds, static asset server logs, and clusters network connectivity."
		latencyDuration := time.Duration(effectiveLoadTimeMs) * time.Millisecond
		go services.Email.SendLatencyAlert("BROWSER", "React UI Load", "rum-telemetry", latencyDuration, c.ClientIP(), reason, action)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "UI load time telemetry recorded successfully",
		"load_time_ms": effectiveLoadTimeMs,
	})
}
