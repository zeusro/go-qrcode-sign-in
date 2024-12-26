package main

import (
	"github.com/gin-gonic/gin"
	_ "go-example/core"
	"go-example/service"
	"log"
	"net/http"
)

func main() {
	//initDB()

	// 初始化 Gin
	r := gin.Default()

	// 设置模板目录
	r.LoadHTMLGlob("templates/*")

	// 静态资源目录
	r.Static("/static", "./static")

	// 首页路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// 生成二维码接口

	r.GET("/generate_qrcode", service.GenerateQRCode)

	// WebSocket 连接处理
	r.GET("/ws", service.WS)

	// 模拟扫码接口
	r.GET("/scan_qrcode", service.ScanQRCode)

	log.Println("Server running on :8080")
	r.Run(":8080")
}
