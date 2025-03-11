package routes

import (
	"github.com/gofiber/fiber/v2"

	"go_creation/handlers"
	"go_creation/middleware"
)

// SetupSalespersonAgentRoutes 设置销售员代理相关的路由
func SetupSalespersonAgentRoutes(app *fiber.App) {
	// 代理相关API路由组
	agentGroup := app.Group("/api/salesperson/agent")

	// 需要销售员身份验证的路由
	agentGroup.Use(middleware.SalespersonAuthMiddleware())

	// 获取代理层级结构
	agentGroup.Get("/hierarchy", handlers.GetAgentHierarchy)

	// 获取代理佣金记录
	agentGroup.Get("/commissions", handlers.GetAgentCommissions)

	// 创建代理邀请
	agentGroup.Post("/invitation", handlers.CreateAgentInvitation)

	// 接受代理邀请
	agentGroup.Post("/invitation/accept", handlers.AcceptAgentInvitation)

	// 生成代理码（管理员操作）
	app.Post("/api/admin/salesperson/:id/agent-code", handlers.GenerateAgentCode)
}
