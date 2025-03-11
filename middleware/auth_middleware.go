package middleware

import (
	"fmt"
	"go_creation/database"
	"go_creation/models"
	"go_creation/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SalespersonAuthMiddleware 验证销售员身份的中间件
// 该中间件负责处理所有需要销售员身份验证的路由请求
// 支持两种认证方式:
//  1. JWT令牌认证 - 通过Authorization头的Bearer令牌
//  2. 兼容模式 - 通过X-Salesperson-ID头直接指定销售员ID（主要用于测试和旧版本兼容）
//
// 认证成功后，会将销售员信息存储在请求上下文中，供后续处理函数使用
// 认证失败则会返回相应的错误信息和状态码
func SalespersonAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 从请求头获取Authorization
		// 检查是否提供了Bearer令牌
		authHeader := c.Get("Authorization")
		fmt.Println("认证中间件 - Authorization头:", authHeader)

		// 如果没有Authorization头，尝试从X-Salesperson-ID获取（用于兼容旧版本和测试）
		// 这是一种备选的认证方式，主要用于API测试和开发环境
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			salespersonIDStr := c.Get("X-Salesperson-ID")
			fmt.Println("认证中间件 - X-Salesperson-ID头:", salespersonIDStr)
			
			if salespersonIDStr == "" {
				fmt.Println("认证中间件 - 未提供有效的认证令牌")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "未提供有效的认证令牌",
				})
			}

			// 将ID转换为整数
			// 验证ID格式是否正确
			salespersonID, err := strconv.Atoi(salespersonIDStr)
			if err != nil {
				fmt.Println("认证中间件 - 无效的销售员ID:", err)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "无效的销售员ID",
				})
			}

			// 查询销售员信息
			// 验证销售员是否存在且状态为活跃
			var salesperson models.Salesperson
			if err := database.GetDB().Where("id = ? AND status = ?", salespersonID, "active").First(&salesperson).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					fmt.Println("认证中间件 - 销售员不存在或已被禁用")
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "销售员不存在或已被禁用",
					})
				}
				fmt.Println("认证中间件 - 验证销售员身份失败:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "验证销售员身份失败",
				})
			}

			// 将销售员信息存储在上下文中，供后续处理函数使用
			// 这些信息可以通过c.Locals()在后续处理函数中获取
			c.Locals("salesperson_id", salesperson.ID)
			c.Locals("salesperson_name", salesperson.Name)
			fmt.Printf("认证中间件 - 通过X-Salesperson-ID认证成功，ID=%d, 名称=%s\n", salesperson.ID, salesperson.Name)

			// 设置请求头，方便后续处理函数使用
			// 确保X-Salesperson-ID头的值与已验证的销售员ID一致
			c.Set("X-Salesperson-ID", strconv.FormatUint(uint64(salesperson.ID), 10))

			// 继续处理请求
			// 认证成功，允许请求继续传递到下一个处理函数
			return c.Next()
		}

		// 从Authorization头中提取令牌
		// 去掉"Bearer "前缀，获取实际的JWT令牌字符串
		tokenString := authHeader[7:] // 去掉"Bearer "前缀
		fmt.Println("认证中间件 - JWT令牌:", tokenString)

		// 解析令牌
		// 验证JWT令牌的签名并提取声明信息
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			fmt.Println("认证中间件 - 解析JWT令牌失败:", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "无效的认证令牌",
			})
		}
		fmt.Printf("认证中间件 - JWT令牌解析成功，销售员ID=%d\n", claims.SalespersonID)

		// 检查令牌是否存在于数据库
		// 确保令牌未被撤销且仍然有效
		var token models.SalespersonToken
		if err := database.GetDB().Where("token = ?", tokenString).First(&token).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				fmt.Println("认证中间件 - 令牌不存在于数据库")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "认证令牌不存在",
				})
			}
			fmt.Println("认证中间件 - 验证令牌失败:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "验证认证令牌失败",
			})
		}

		// 检查令牌是否已过期
		// 即使JWT本身未过期，也需检查数据库中的过期时间
		if time.Now().After(token.ExpiredAt) {
			fmt.Println("认证中间件 - 令牌已过期")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "认证令牌已过期",
			})
		}

		// 查询销售员信息
		// 验证销售员是否存在且状态为活跃
		var salesperson models.Salesperson
		if err := database.GetDB().Where("id = ? AND status = ?", claims.SalespersonID, "active").First(&salesperson).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				fmt.Println("认证中间件 - 销售员不存在或已被禁用")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "销售员不存在或已被禁用",
				})
			}
			fmt.Println("认证中间件 - 验证销售员身份失败:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "验证销售员身份失败",
			})
		}

		// 将销售员信息存储在上下文中，供后续处理函数使用
		// 这些信息可以通过c.Locals()在后续处理函数中获取
		c.Locals("salesperson_id", salesperson.ID)
		c.Locals("salesperson_name", salesperson.Name)
		fmt.Printf("认证中间件 - 通过JWT认证成功，ID=%d, 名称=%s\n", salesperson.ID, salesperson.Name)

		// 设置请求头，方便后续处理函数使用
		// 确保X-Salesperson-ID头的值与已验证的销售员ID一致
		c.Set("X-Salesperson-ID", strconv.FormatUint(uint64(salesperson.ID), 10))

		// 继续处理请求
		// 认证成功，允许请求继续传递到下一个处理函数
		return c.Next()
	}
}
