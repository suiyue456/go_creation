package routes

import (
	"go_creation/handlers"
	"go_creation/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterKeyRoutes 设置卡密相关路由
func RegisterKeyRoutes(api fiber.Router) {
	// 卡密相关路由
	keys := api.Group("/keys")
	
	// 不需要认证的路由 - 必须放在前面，避免被认证中间件拦截
	keys.Post("/activate", handlers.ActivateKey)  // 激活卡密
	keys.Get("/status", handlers.GetKeyStatus)    // 查询卡密状态
	
	// 需要认证的路由
	authKeys := keys.Group("/", middleware.SalespersonAuthMiddleware())
	authKeys.Post("/batch", handlers.BatchCreateKeys) // 批量创建卡密
	authKeys.Get("/", handlers.GetAllKeys)            // 获取所有卡密
	authKeys.Get("/:id", handlers.GetKeyByID)         // 获取单个卡密
	authKeys.Put("/:id/void", handlers.VoidKey)       // 作废卡密
	authKeys.Get("/export", handlers.ExportKeys)      // 导出卡密

	// 软件卡密相关路由 - 需要认证
	api.Get("/software/:id/keys", middleware.SalespersonAuthMiddleware(), handlers.GetKeysBySoftwareID) // 按软件ID查询卡密
}
