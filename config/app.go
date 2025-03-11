// Package config 提供应用程序配置和初始化功能
// 该包负责处理应用程序的配置加载、初始化和服务器设置等核心功能
package config

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"go_creation/database"
	"go_creation/routes"
)

// InitApp 初始化整个应用程序
// 该函数是应用程序启动的核心，负责：
// 1. 初始化数据库连接
// 2. 执行数据库迁移
// 3. 设置全局配置
// 4. 初始化必要的服务
func InitApp() {
	// 初始化数据库连接
	// 如果数据库连接失败，程序将终止
	database.Init()

	// 执行数据库迁移
	// 确保所有必要的表和结构都存在
	database.Migrate()

	log.Println("应用程序初始化完成")
}

// SetupApp 创建并配置Fiber应用实例
// 该函数负责：
// 1. 创建新的Fiber实例
// 2. 配置全局中间件
// 3. 设置路由
// 4. 配置错误处理
// 返回配置完成的Fiber实例
func SetupApp() *fiber.App {
	// 创建新的Fiber实例
	// 配置自定义的错误处理
	app := fiber.New(fiber.Config{
		// 启用案例敏感的路由
		CaseSensitive: true,
		// 严格的路由规则
		StrictRouting: true,
		// 服务器名称
		ServerHeader: "Go Creation",
		// 限制请求体大小为10MB
		BodyLimit: 10 * 1024 * 1024,
		// 自定义错误处理
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// 默认错误码为500
			code := fiber.StatusInternalServerError

			// 如果是Fiber的错误，使用其状态码
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			// 构建错误响应
			return c.Status(code).JSON(fiber.Map{
				"error": true,
				"msg":   err.Error(),
			})
		},
		// 添加以下配置解决中文编码问题
		JSONEncoder: json.Marshal,   // 使用标准JSON编码器，确保正确处理UTF-8字符
		JSONDecoder: json.Unmarshal, // 使用标准JSON解码器，确保正确解析UTF-8字符
		Immutable:   true,           // 设置响应头不可变，提高性能
		// 设置应用名称和其他配置
		AppName:      "Go Creation API",
		ReadTimeout:  60 * time.Second, // 读取超时时间，防止慢客户端攻击
		WriteTimeout: 60 * time.Second, // 写入超时时间，确保响应能够及时完成
		IdleTimeout:  60 * time.Second, // 空闲超时时间，优化连接池使用
	})

	// 配置日志中间件
	// 记录所有HTTP请求111
	app.Use(logger.New(logger.Config{
		// 自定义日志格式
		Format: "${time} ${status} - ${method} ${path}\n",
		// 日志时间格式
		TimeFormat: "2006-01-02 15:04:05",
		// 日志输出位置
		Output: os.Stdout,
	}))

	// 配置恢复中间件
	// 防止应用因panic而崩溃
	app.Use(recover.New())

	// 配置CORS中间件
	// 允许跨域请求
	app.Use(cors.New(cors.Config{
		// 允许的源
		AllowOrigins: "*",
		// 允许的方法
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		// 允许的头部
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		// 允许携带认证信息
		AllowCredentials: true,
		// 预检请求的有效期
		MaxAge: int(12 * time.Hour.Seconds()),
	}))

	// 设置API路由
	// 所有的API路由都以/api为前缀
	routes.SetupRoutes(app)

	log.Println("Fiber应用已创建，使用UTF-8编码")
	log.Println("路由已设置")

	return app
}
