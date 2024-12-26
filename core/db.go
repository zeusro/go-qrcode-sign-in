package core

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"time"
)

// 数据库连接
var DB *gorm.DB

func init() {
	dsn := "root:root@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	DB.AutoMigrate(&QRCode{})
}

// 数据库模型
type QRCode struct {
	ID        uint   `gorm:"primaryKey"`
	Code      string `gorm:"unique;not null"`
	Status    string `gorm:"default:pending"`
	UserID    *uint  `gorm:"default:null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
