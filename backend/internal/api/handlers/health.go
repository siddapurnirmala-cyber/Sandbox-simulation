package handlers

import (
	"net/http"

	"backend/internal/database"

	"github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "DOWN", "reason": "Database connection not initialized"})
		return
	}

	sqlDB, err := database.DB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "DOWN", "reason": "Failed to get database handle", "error": err.Error()})
		return
	}

	err = sqlDB.Ping()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "DOWN", "reason": "Database connection lost", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}
