package routes

import (
	// "go_creation/handlers"
	"go_creation/handlers"
	"go_creation/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupSalespersonRoutes 设置销售员相关的路由
func SetupSalespersonRoutes(app *fiber.App) {
	// 销售员管理路由组（管理员访问）
	salespersonGroup := app.Group("/api/salespersons")

	//销售员基本管理
	salespersonGroup.Post("/", handlers.CreateSalesperson)      // 创建销售员
	salespersonGroup.Get("/", handlers.GetAllSalespersons)      // 获取所有销售员
	salespersonGroup.Get("/:id", handlers.GetSalesperson)       // 获取单个销售员
	salespersonGroup.Put("/:id", handlers.UpdateSalesperson)    // 更新销售员
	salespersonGroup.Delete("/:id", handlers.DeleteSalesperson) // 删除销售员

	// 销售员登录
	app.Post("/api/salesperson/login", handlers.SalespersonLogin) // 销售员登录

	// 销售员产品管理（管理员访问）
	salespersonGroup.Get("/:id/products", handlers.GetSalespersonProducts)     // 获取销售员可销售的产品
	app.Post("/api/salesperson-products", handlers.AssignProductToSalesperson) // 为销售员分配产品

	// 销售员销售记录（管理员访问）
	salespersonGroup.Get("/:id/sales", handlers.GetSalespersonSales)           // 获取销售员的销售记录
	salespersonGroup.Get("/:id/commission", handlers.GetSalespersonCommission) // 获取销售员的佣金统计

	// 销售员专用API（需要销售员身份验证）
	salespersonAPI := app.Group("/api/salesperson", middleware.SalespersonAuthMiddleware())

	// 销售员卡密生成
	salespersonAPI.Post("/generate-keys", handlers.GenerateKeysForSalesperson) // 销售员生成卡密

	// 销售员查询自己的产品
	salespersonAPI.Get("/products", handlers.GetSalespersonOwnProducts) // 获取销售员自己可销售的产品

	// 销售员查询自己的销售记录
	salespersonAPI.Get("/sales", handlers.GetSalespersonOwnSales) // 获取销售员自己的销售记录

	// 销售员查询自己的佣金
	salespersonAPI.Get("/commission", handlers.GetSalespersonOwnCommission) // 获取销售员自己的佣金统计
}
