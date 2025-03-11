package routes

import (
	"go_creation/handlers"
	"go_creation/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupAuthRoutes 设置认证相关路由
// 该函数负责注册所有与认证相关的API路由，包括登录、登出、刷新令牌等功能
// 认证系统采用JWT（JSON Web Token）机制，支持多设备登录和会话管理
// 路由组织结构遵循RESTful API设计原则，使用适当的HTTP方法和路径
func SetupAuthRoutes(app *fiber.App) {
	// 创建认证相关的路由组，所有认证相关的路由都将以/api/auth为前缀
	// 这种分组方式有助于API的组织和版本管理
	auth := app.Group("/api/auth")

	// 登录路由 - 处理销售员的登录请求
	// POST /api/auth/login
	// 请求体需包含用户名和密码
	// 成功返回JWT令牌和过期时间
	// 不需要认证中间件，因为用户尚未登录
	auth.Post("/login", handlers.SalespersonLogin)

	// 登出路由 - 处理销售员的登出请求
	// POST /api/auth/logout
	// 使当前会话的令牌失效
	// 需要认证中间件确保用户已登录
	auth.Post("/logout", middleware.SalespersonAuthMiddleware(), handlers.SalespersonLogout)

	// 刷新令牌路由 - 用于刷新JWT令牌，延长登录有效期
	// POST /api/auth/refresh
	// 使用当前令牌获取新令牌，避免用户频繁登录
	// 不需要认证中间件，因为令牌可能已过期，但仍可用于刷新
	auth.Post("/refresh", handlers.RefreshToken)

	// 获取登录设备列表路由 - 查询当前销售员的所有登录设备
	// GET /api/auth/devices
	// 返回所有活跃的登录会话信息，包括设备类型、IP地址和登录时间
	// 需要认证中间件确保用户已登录
	auth.Get("/devices", middleware.SalespersonAuthMiddleware(), handlers.GetLoginDevices)

	// 登出特定设备路由 - 使特定设备的登录会话失效
	// DELETE /api/auth/devices/:id
	// 路径参数id指定要登出的设备ID
	// 允许用户管理自己的多设备登录状态
	// 需要认证中间件确保用户已登录
	auth.Delete("/devices/:id", middleware.SalespersonAuthMiddleware(), handlers.LogoutDevice)

	// 强制登出销售员路由 - 管理员功能，使指定销售员的所有登录会话失效
	// DELETE /api/auth/salesperson/:id/logout
	// 路径参数id指定要强制登出的销售员ID
	// 用于账户安全管理，如检测到异常登录活动时
	// 需要认证中间件，实际应用中应该使用管理员认证中间件
	auth.Delete("/salesperson/:id/logout", middleware.SalespersonAuthMiddleware(), handlers.ForceLogoutSalesperson)
}
