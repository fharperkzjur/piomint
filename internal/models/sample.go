
package models

import (
	"gorm.io/gorm"
)

type Sample struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt UnixTime
	UpdatedAt UnixTime
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
