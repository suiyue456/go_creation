package routes

import (
	"go_creation/handlers"

	"github.com/gofiber/fiber/v2"
)

// RegisterKeyTypeRoutes 设置卡密类型相关路由
func RegisterKeyTypeRoutes(api fiber.Router) {
	// 卡密类型相关路由
	keyTypes := api.Group("/keytypes")
	keyTypes.Post("/", handlers.CreateKeyType)                   // 创建卡密类型
	keyTypes.Get("/", handlers.GetAllKeyTypes)                   // 获取所有卡密类型
	keyTypes.Get("/:id", handlers.GetKeyTypeByID)                // 获取单个卡密类型
	keyTypes.Put("/:id", handlers.UpdateKeyType)                 // 更新卡密类型
	keyTypes.Delete("/:id", handlers.DeleteKeyType)              // 删除卡密类型
	keyTypes.Post("/:id/activate", handlers.ActivateKeyType)     // 激活卡密类型
	keyTypes.Post("/:id/deactivate", handlers.DeactivateKeyType) // 停用卡密类型
}
