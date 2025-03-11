package routes

import (
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes 设置所有API路由
// 调用各个模块的路由注册函数
func SetupRoutes(app *fiber.App) {
	// API路由组
	api := app.Group("/api")

	// 设置各模块路由
	RegisterKeyRoutes(api)
	RegisterKeyTypeRoutes(api)
	RegisterSoftwareRoutes(api)

	// 设置销售员路由
	SetupSalespersonRoutes(app)

	// 设置销售员代理路由
	SetupSalespersonAgentRoutes(app)

	// 设置认证路由
	SetupAuthRoutes(app)
}
