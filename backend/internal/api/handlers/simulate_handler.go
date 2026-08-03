package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"backend/internal/api/models"
	"backend/internal/api/services"
	"backend/internal/database"
	"backend/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DelayInput struct {
	DelayMs int `json:"delay_ms"`
}

type EnableInput struct {
	Enable bool `json:"enable"`
}

type MemoryInput struct {
	Megabytes int   `json:"megabytes"`
	Enable    *bool `json:"enable"`
}

// POST /simulate/api-delay
func SimulateAPIDelay(c *gin.Context) {
	var input DelayInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.FailureConfig.SetAPIDelay(input.DelayMs)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Simulated API delay updated successfully",
		"delay_ms": input.DelayMs,
	})
}

// POST /simulate/db-delay
func SimulateDBDelay(c *gin.Context) {
	var input DelayInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.FailureConfig.SetDBDelay(input.DelayMs)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Simulated DB delay updated successfully",
		"delay_ms": input.DelayMs,
	})
}

// POST /simulate/db-failure
func SimulateDBFailure(c *gin.Context) {
	var input EnableInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.FailureConfig.SetDBFailure(input.Enable)
	c.JSON(http.StatusOK, gin.H{
		"message": "Simulated DB failure state changed",
		"enabled": input.Enable,
	})
}

// POST /simulate/vsi-timeout
func SimulateVSITimeout(c *gin.Context) {
	var input EnableInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.FailureConfig.SetVSITimeout(input.Enable)
	c.JSON(http.StatusOK, gin.H{
		"message": "Simulated VSI connection timeout state changed",
		"enabled": input.Enable,
	})
}

// POST /simulate/random-errors
func SimulateRandomErrors(c *gin.Context) {
	var input EnableInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.FailureConfig.SetRandomErrors(input.Enable)
	c.JSON(http.StatusOK, gin.H{
		"message": "Simulated random HTTP API errors changed",
		"enabled": input.Enable,
	})
}

// POST /simulate/high-memory
func SimulateHighMemory(c *gin.Context) {
	var input MemoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Release memory if Enable is explicitly false
	if input.Enable != nil && !*input.Enable {
		services.FailureConfig.TriggerMemoryAllocation(0)
		c.JSON(http.StatusOK, gin.H{
			"message": "Simulated memory leak cleared",
			"leak_mb": 0,
		})
		return
	}

	services.FailureConfig.TriggerMemoryAllocation(input.Megabytes)
	c.JSON(http.StatusOK, gin.H{
		"message": "Simulated memory leak triggered",
		"leak_mb": input.Megabytes,
	})
}

// POST /simulate/high-cpu
func SimulateHighCPU(c *gin.Context) {
	var input EnableInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.FailureConfig.SetCPUBurn(input.Enable)
	c.JSON(http.StatusOK, gin.H{
		"message": "Simulated high CPU state changed",
		"enabled": input.Enable,
	})
}

// GET /simulate/api-delay/:ms
func SimulateAPIDelayGet(c *gin.Context) {
	msStr := c.Param("ms")
	ms, err := strconv.Atoi(msStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid delay value. Must be an integer."})
		return
	}

	services.FailureConfig.SetAPIDelay(ms)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Simulated API delay updated successfully via GET request",
		"delay_ms": ms,
	})
}

// GET /simulate/db-delay/:ms
func SimulateDBDelayGet(c *gin.Context) {
	msStr := c.Param("ms")
	ms, err := strconv.Atoi(msStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid delay value. Must be an integer."})
		return
	}

	services.FailureConfig.SetDBDelay(ms)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Simulated DB delay updated successfully via GET request",
		"delay_ms": ms,
	})
}

// POST /simulate/ui-delay
func SimulateUIDelay(c *gin.Context) {
	var input DelayInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.FailureConfig.SetUIDelay(input.DelayMs)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Simulated UI delay updated successfully",
		"delay_ms": input.DelayMs,
	})
}

// GET /simulate/ui-delay/:ms
func SimulateUIDelayGet(c *gin.Context) {
	msStr := c.Param("ms")
	ms, err := strconv.Atoi(msStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid delay value. Must be an integer."})
		return
	}

	services.FailureConfig.SetUIDelay(ms)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Simulated UI delay updated successfully via GET request",
		"delay_ms": ms,
	})
}

// GET /simulate/ui-delay
func GetUIDelay(c *gin.Context) {
	services.FailureConfig.Lock()
	ms := services.FailureConfig.UIDelay.Milliseconds()
	services.FailureConfig.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"delay_ms": ms,
	})
}

// POST /simulate/heavy-load
func SimulateHeavyLoad(c *gin.Context) {
	var input EnableInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.FailureConfig.Lock()
	currentHeavyLoadState := services.FailureConfig.HeavyLoad
	services.FailureConfig.Unlock()

	if input.Enable {
		if !currentHeavyLoadState {
			if err := generateHeavyLogs(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate heavy logs: " + err.Error()})
				return
			}
			services.FailureConfig.SetHeavyLoad(true)
		}
	} else {
		if currentHeavyLoadState {
			if err := clearHeavyLogs(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear heavy logs: " + err.Error()})
				return
			}
			services.FailureConfig.SetHeavyLoad(false)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Simulated Heavy Load state changed successfully",
		"enabled": input.Enable,
	})
}

func generateHeavyLogs() error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}

	logger.Log.Info("Generating 50000 simulated heavy load logs in Postgres...")

	totalLogs := 50000
	batchSize := 5000
	logs := make([]models.SandboxLog, totalLogs)
	now := time.Now()

	for i := 0; i < totalLogs; i++ {
		logs[i] = models.SandboxLog{
			SandboxID: 9999,
			Message:   fmt.Sprintf("Simulated heavy data audit log entry #%d. This is to test DB and UI pagination capacity.", i),
			LogLevel:  "INFO",
			CreatedAt: now.Add(time.Duration(-i) * time.Second),
		}
	}

	err := database.DB.CreateInBatches(logs, batchSize).Error
	if err != nil {
		logger.Log.Error("Failed to insert heavy load logs", zap.Error(err))
		return err
	}

	logger.Log.Info("Successfully inserted 50000 heavy load logs")
	return nil
}

func clearHeavyLogs() error {
	if database.DB == nil {
		return errors.New("database not initialized")
	}

	logger.Log.Info("Clearing all heavy load logs from database...")
	err := database.DB.Where("sandbox_id = ?", 9999).Delete(&models.SandboxLog{}).Error
	if err != nil {
		logger.Log.Error("Failed to clear heavy load logs", zap.Error(err))
		return err
	}

	logger.Log.Info("Successfully cleared heavy load logs")
	return nil
}

// GET /simulate/states
func GetSimulationStates(c *gin.Context) {
	services.FailureConfig.Lock()
	defer services.FailureConfig.Unlock()

	cpuActive := len(services.FailureConfig.CpuBurnCancel) > 0

	memSize := 0
	for _, block := range services.FailureConfig.MemoryLeak {
		memSize += len(block) / (1024 * 1024)
	}

	c.JSON(http.StatusOK, gin.H{
		"api_delay_ms":  services.FailureConfig.APIDelay.Milliseconds(),
		"db_delay_ms":   services.FailureConfig.DBDelay.Milliseconds(),
		"db_failure":    services.FailureConfig.DBFailure,
		"ui_delay_ms":   services.FailureConfig.UIDelay.Milliseconds(),
		"vsi_timeout":   services.FailureConfig.VSITimeout,
		"random_errors": services.FailureConfig.RandomErrors,
		"heavy_load":    services.FailureConfig.HeavyLoad,
		"high_cpu":      cpuActive,
		"high_memory":   memSize,
	})
}
