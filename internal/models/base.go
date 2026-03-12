package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== BASE ====================

type BaseModel struct {
	ID        uuid.UUID      `gorm:"primaryKey;type:uuid;default:gen_random_uuid();column:id"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index;column:deleted_at"`
}

type JSONMap map[string]interface{}
