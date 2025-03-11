package routes

import (
	"go_creation/handlers"

	"github.com/gofiber/fiber/v2"
)

// RegisterSoftwareRoutes 设置软件相关路由
func RegisterSoftwareRoutes(api fiber.Router) {
	// 软件管理路由
	software := api.Group("/software")
	software.Post("/", handlers.CreateSoftware)                  // 创建软件
	software.Get("/", handlers.GetAllSoftware)                   // 获取所有软件
	software.Get("/:id", handlers.GetSoftwareByID)               // 获取单个软件
	software.Put("/:id", handlers.UpdateSoftware)                // 更新软件
	software.Delete("/:id", handlers.DeleteSoftware)             // 删除软件
	software.Put("/:id/activate", handlers.ActivateSoftware)     // 激活软件
	software.Put("/:id/deactivate", handlers.DeactivateSoftware) // 停用软件
	software.Get("/:id/keytypes", handlers.GetSoftwareKeyTypes)  // 获取软件绑定的卡密类型
	software.Post("/bind-keytype", handlers.BindKeyType)         // 绑定卡密类型
	software.Post("/unbind-keytype", handlers.UnbindKeyType)     // 解绑卡密类型
}
