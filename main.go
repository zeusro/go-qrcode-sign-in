package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// WebSocket 升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
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

// 数据库连接
var db *gorm.DB
var clients = make(map[string]*websocket.Conn) // 保存客户端 WebSocket 连接
var mutex sync.Mutex

func initDB() {
	dsn := "root:root@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	db.AutoMigrate(&QRCode{})
}

// 模拟扫码接口
func scanQRCode(c *gin.Context) {
	code := c.Query("code")
	var qr QRCode
	if err := db.Where("code = ?", code).First(&qr).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid code"})
		return
	}

	// 更新状态
	db.Model(&qr).Update("status", "scanned")

	// 通知前端
	mutex.Lock()
	if conn, ok := clients[code]; ok {
		conn.WriteJSON(map[string]string{
			"status": "登录成功",
		})
		conn.Close()
		delete(clients, code)
	}
	mutex.Unlock()

	c.Status(http.StatusOK)
}

func main() {
	initDB()

	// 初始化 Gin
	r := gin.Default()

	// 设置模板目录
	r.LoadHTMLGlob("templates/*")

	// 静态资源目录
	//r.Static("/static", "./static")

	// 首页路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.GET("/q", func(c *gin.Context) {
		c.HTML(http.StatusOK, "q.html", nil)
	})

	// 生成二维码接口

	r.GET("/generate_qrcode", generateQRCode)

	// WebSocket 连接处理
	r.GET("/ws", ws)

	// 模拟扫码接口
	r.GET("/scan_qrcode", scanQRCode)

	log.Println("Server running on :8080")
	r.Run(":8080")
}

func ws(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// 监听客户端消息
	var msg struct {
		Code string `json:"code"`
	}
	if err := conn.ReadJSON(&msg); err != nil {
		log.Printf("Error reading WebSocket message: %v", err)
		return
	}

	// 保存 WebSocket 连接
	mutex.Lock()
	clients[msg.Code] = conn
	mutex.Unlock()

	// 等待状态变化
	for {
		var qr QRCode
		if err := db.Where("code = ?", msg.Code).First(&qr).Error; err != nil {
			log.Printf("Error querying QR code: %v", err)
			return
		}

		if qr.Status != "pending" {
			conn.WriteJSON(map[string]string{
				"status": qr.Status,
			})
			return
		}
		time.Sleep(1 * time.Second)
	}
}

// 生成二维码并返回
func generateQRCode(c *gin.Context) {
	code := fmt.Sprintf("%d", time.Now().UnixNano())
	qr := QRCode{
		Code: code,
	}
	db.Create(&qr)

	c.JSON(http.StatusOK, gin.H{
		"qrcode": code,
	})
}

func generateQRCodeImage(c *gin.Context) {
	code := fmt.Sprintf("%d", time.Now().UnixNano())
	qr := QRCode{
		Code: code,
	}
	db.Create(&qr)

	// 获取请求参数 (二维码的内容)
	u := fmt.Sprintf("http://localhost:8080/scan_qrcode", code)
	content := c.DefaultQuery("content", u)

	// 生成二维码
	qrCode, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}

	// 返回二维码图片
	c.Header("Content-Type", "image/png")
	c.Writer.Write(qrCode)
}
