package handlers

import (
	"net/http"
	"strconv"

	"backend/internal/api/models"
	"backend/internal/api/services"
	"backend/internal/database"
	"backend/internal/logger"
	"backend/internal/metrics"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CreateSandboxInput struct {
	Name  string `json:"sandbox_name" binding:"required"`
	Owner string `json:"owner" binding:"required"`
}

type CommandInput struct {
	Command string `json:"command" binding:"required"`
}

// POST /sandbox
func CreateSandbox(c *gin.Context) {
	var input CreateSandboxInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sandbox := models.Sandbox{
		SandboxName: input.Name,
		Owner:       input.Owner,
		Status:      "STOPPED",
	}

	if err := database.DB.Create(&sandbox).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sandbox"})
		return
	}

	metrics.SandboxCreatedTotal.Inc()

	// Write audit log
	auditLog := models.SandboxLog{
		SandboxID: sandbox.ID,
		Message:   "Sandbox created and initialized in STOPPED state.",
		LogLevel:  "INFO",
	}
	_ = database.DB.Create(&auditLog)

	c.JSON(http.StatusCreated, sandbox)
}

// GET /sandbox
func ListSandboxes(c *gin.Context) {
	var sandboxes []models.Sandbox
	if err := database.DB.Find(&sandboxes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query sandboxes"})
		return
	}

	c.JSON(http.StatusOK, sandboxes)
}

// GET /sandbox/:id
func GetSandbox(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sandbox ID"})
		return
	}

	var sandbox models.Sandbox
	if err := database.DB.First(&sandbox, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sandbox not found"})
		return
	}

	c.JSON(http.StatusOK, sandbox)
}

// DELETE /sandbox/:id
func DeleteSandbox(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sandbox ID"})
		return
	}

	var sandbox models.Sandbox
	if err := database.DB.First(&sandbox, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sandbox not found"})
		return
	}

	// Delete sandbox logs first
	_ = database.DB.Where("sandbox_id = ?", uint(id)).Delete(&models.SandboxLog{})

	if err := database.DB.Delete(&sandbox).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete sandbox"})
		return
	}

	metrics.SandboxDeletedTotal.Inc()

	c.JSON(http.StatusOK, gin.H{"message": "Sandbox and its logs deleted successfully"})
}

// POST /sandbox/:id/connect
func ConnectSandbox(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sandbox ID"})
		return
	}

	var sandbox models.Sandbox
	if err := database.DB.First(&sandbox, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sandbox not found"})
		return
	}

	// Update status to PENDING during connection attempt
	sandbox.Status = "PENDING"
	_ = database.DB.Save(&sandbox)

	connTime, err := services.VSI.Connect(uint(id))

	var auditLog models.SandboxLog
	auditLog.SandboxID = uint(id)

	if err != nil {
		sandbox.Status = "ERROR"
		_ = database.DB.Save(&sandbox)

		auditLog.Message = "VSI connection failed: " + err.Error()
		auditLog.LogLevel = "ERROR"
		_ = database.DB.Create(&auditLog)

		logger.Log.Error("Sandbox connection failed", zap.Uint("sandbox_id", uint(id)), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":           "Failed to establish connection to virtual instance",
			"details":         err.Error(),
			"connection_time": connTime.String(),
		})
		return
	}

	sandbox.Status = "RUNNING"
	_ = database.DB.Save(&sandbox)

	auditLog.Message = "VSI connection established successfully in " + connTime.String()
	auditLog.LogLevel = "INFO"
	_ = database.DB.Create(&auditLog)

	c.JSON(http.StatusOK, gin.H{
		"message":         "Sandbox connected successfully",
		"status":          "RUNNING",
		"connection_time": connTime.String(),
	})
}

// POST /sandbox/:id/disconnect
func DisconnectSandbox(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sandbox ID"})
		return
	}

	var sandbox models.Sandbox
	if err := database.DB.First(&sandbox, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sandbox not found"})
		return
	}

	sandbox.Status = "STOPPED"
	_ = database.DB.Save(&sandbox)

	// Write log
	auditLog := models.SandboxLog{
		SandboxID: uint(id),
		Message:   "VSI disconnected.",
		LogLevel:  "INFO",
	}
	_ = database.DB.Create(&auditLog)

	logger.Log.Info("Sandbox disconnected", zap.Uint("sandbox_id", uint(id)))

	c.JSON(http.StatusOK, gin.H{
		"message": "Sandbox disconnected successfully",
		"status":  "STOPPED",
	})
}

// POST /sandbox/:id/run-command
func RunCommandSandbox(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sandbox ID"})
		return
	}

	var sandbox models.Sandbox
	if err := database.DB.First(&sandbox, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sandbox not found"})
		return
	}

	if sandbox.Status != "RUNNING" {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot run command. Sandbox is not in RUNNING status"})
		return
	}

	var input CommandInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := services.VSI.RunCommand(uint(id), input.Command)

	var auditLog models.SandboxLog
	auditLog.SandboxID = uint(id)

	if err != nil {
		auditLog.Message = "VSI command run failed ('" + input.Command + "'): " + err.Error()
		auditLog.LogLevel = "WARNING"
		_ = database.DB.Create(&auditLog)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Command execution failed",
			"details": err.Error(),
		})
		return
	}

	auditLog.Message = "VSI command executed: '" + input.Command + "'"
	auditLog.LogLevel = "INFO"
	_ = database.DB.Create(&auditLog)

	c.JSON(http.StatusOK, gin.H{
		"command": input.Command,
		"output":  output,
	})
}

// GET /logs
func GetLogs(c *gin.Context) {
	var logs []models.SandboxLog

	tx := database.DB.Order("created_at desc")

	// Filter by sandbox ID if requested
	sandboxIDStr := c.Query("sandbox_id")
	if sandboxIDStr != "" {
		if sandboxID, err := strconv.ParseUint(sandboxIDStr, 10, 32); err == nil {
			tx = tx.Where("sandbox_id = ?", uint(sandboxID))
		}
	}

	// Filter by log level if requested
	level := c.Query("level")
	if level != "" {
		tx = tx.Where("log_level = ?", level)
	}

	// Limit to last 100 entries for safety
	tx = tx.Limit(100)

	if err := tx.Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}
