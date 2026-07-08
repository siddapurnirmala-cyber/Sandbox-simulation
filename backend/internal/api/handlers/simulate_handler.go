package handlers

import (
	"net/http"

	"backend/internal/api/services"

	"github.com/gin-gonic/gin"
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
