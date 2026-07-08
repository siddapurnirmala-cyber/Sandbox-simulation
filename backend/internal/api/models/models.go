package models

import (
	"time"
)

type Sandbox struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	SandboxName string    `gorm:"type:varchar(100);not null" json:"sandbox_name"`
	Owner       string    `gorm:"type:varchar(100);not null" json:"owner"`
	Status      string    `gorm:"type:varchar(50);not null;default:'STOPPED'" json:"status"` // PENDING, RUNNING, STOPPED, ERROR
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SandboxLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	SandboxID uint      `gorm:"not null;index" json:"sandbox_id"`
	Message   string    `gorm:"type:text;not null" json:"message"`
	LogLevel  string    `gorm:"type:varchar(20);not null" json:"log_level"` // INFO, WARNING, ERROR
	CreatedAt time.Time `json:"created_at"`
}
