package service

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
	"go-example/core"
	"log"
	"net/http"
	"sync"
	"time"
)

// WebSocket 升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
var clients = make(map[string]*websocket.Conn) // 保存客户端 WebSocket 连接
var mutex sync.Mutex

// 模拟扫码接口
func ScanQRCode(c *gin.Context) {
	code := c.Query("code")
	var qr core.QRCode
	if err := core.DB.Where("code = ?", code).First(&qr).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid code"})
		return
	}

	// 更新状态
	core.DB.Model(&qr).Update("status", "scanned")

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

func WS(c *gin.Context) {
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
		var qr core.QRCode
		if err := core.DB.Where("code = ?", msg.Code).First(&qr).Error; err != nil {
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
func GenerateQRCode(c *gin.Context) {
	code := fmt.Sprintf("%d", time.Now().UnixNano())
	qr := core.QRCode{
		Code: code,
	}
	core.DB.Create(&qr)

	c.JSON(http.StatusOK, gin.H{
		"qrcode": code,
	})
}

func GenerateQRCodeImage(c *gin.Context) {
	code := fmt.Sprintf("%d", time.Now().UnixNano())
	qr := core.QRCode{
		Code: code,
	}
	core.DB.Create(&qr)

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
