// Package config 提供应用程序配置和初始化功能
package config

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
)

// GetPort 获取服务器监听端口
// 该函数从环境变量中读取PORT配置，如果未设置则使用默认端口3000
// 这允许在不同环境（开发、测试、生产）中灵活配置服务端口
func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // 默认端口
		log.Println("未设置PORT环境变量，使用默认端口:", port)
	}
	return port
}

// StartServer 启动HTTP服务器并处理优雅关闭
// 该函数负责：
// 1. 从环境变量获取服务器配置
// 2. 启动HTTP服务器
// 3. 监听系统信号
// 4. 处理优雅关闭
// 参数：
//   - app: 配置好的Fiber应用实例
func StartServer(app *fiber.App) {
	// 从环境变量获取服务器端口
	// 如果未设置，默认使用8080
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	// 创建系统信号通道
	// 用于接收操作系统的终止信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在单独的goroutine中启动服务器
	// 这样可以在主goroutine中处理信号
	go func() {
		// 启动HTTP服务器
		// 如果启动失败，记录错误并终止程序
		if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	log.Printf("服务器已启动，监听端口 %s", port)

	// 等待系统信号
	<-sigChan
	log.Println("收到终止信号，开始优雅关闭...")

	// 优雅关闭服务器
	// 确保所有活跃的连接都能正常完成
	if err := app.Shutdown(); err != nil {
		log.Printf("服务器关闭时发生错误: %v", err)
	}

	log.Println("服务器已安全关闭")
}
