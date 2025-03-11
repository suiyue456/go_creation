package handlers

import (
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go_creation/database"
	"go_creation/models"
	"go_creation/utils"
)

// RefreshToken 刷新认证令牌
// 该处理函数用于延长用户会话，通过验证现有令牌并签发新令牌
// 处理流程:
//  1. 从请求头提取当前令牌
//  2. 验证令牌的有效性和存在性
//  3. 检查令牌是否过期
//  4. 验证关联销售员的状态
//  5. 生成新令牌并存储到数据库
//  6. 删除旧令牌
func RefreshToken(c *fiber.Ctx) error {
	// 从请求头获取令牌
	// 验证Authorization头是否存在且格式正确（Bearer格式）
	authHeader := c.Get("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "未提供有效的认证令牌",
		})
	}

	tokenString := authHeader[7:]

	// 解析令牌
	// 使用JWT工具验证令牌签名并提取声明信息
	claims, err := utils.ParseToken(tokenString)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "无效的认证令牌",
		})
	}

	// 检查令牌是否存在于数据库
	// 确保令牌未被撤销且仍然有效
	var token models.SalespersonToken
	if err := database.GetDB().Where("token = ?", tokenString).First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "认证令牌不存在",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "验证认证令牌失败",
		})
	}

	// 检查令牌是否已过期
	// 即使JWT本身未过期，也需检查数据库中的过期时间
	if time.Now().After(token.ExpiredAt) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "认证令牌已过期",
		})
	}

	// 查询销售员信息
	// 验证销售员是否存在且状态为活跃
	var salesperson models.Salesperson
	if err := database.GetDB().Where("id = ? AND status = ?", claims.SalespersonID, "active").First(&salesperson).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "销售员不存在或已被禁用",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "验证销售员身份失败",
		})
	}

	// 懒惰删除：清理该用户的过期令牌
	// 提高系统性能并减少数据库中的无效记录
	if err := database.GetDB().Where("salesperson_id = ? AND expired_at < ?", salesperson.ID, time.Now()).Delete(&models.SalespersonToken{}).Error; err != nil {
		log.Printf("删除过期令牌失败: %v", err)
		// 不返回错误，继续处理
	}

	// 生成新的JWT令牌
	// 设置24小时的有效期
	newToken, err := utils.GenerateToken(salesperson.ID, salesperson.Username, 24*time.Hour)
	if err != nil {
		log.Printf("生成令牌失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "刷新令牌失败，请稍后重试",
		})
	}

	// 计算过期时间
	expireTime := time.Now().Add(24 * time.Hour)

	// 删除旧令牌
	// 确保旧令牌不能再被使用，防止令牌重放攻击
	if err := database.GetDB().Delete(&token).Error; err != nil {
		log.Printf("删除旧令牌失败: %v", err)
		// 不返回错误，继续处理
	}

	// 存储新令牌到数据库
	// 记录令牌信息，包括关联的销售员、设备信息和过期时间
	newSalespersonToken := models.SalespersonToken{
		SalespersonID: salesperson.ID,
		Token:         newToken,
		UserAgent:     token.UserAgent, // 保持原有的用户代理信息
		IP:            c.IP(),          // 更新IP地址
		ExpiredAt:     expireTime,
	}

	if err := database.GetDB().Create(&newSalespersonToken).Error; err != nil {
		log.Printf("存储新令牌失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "刷新令牌失败，请稍后重试",
		})
	}

	// 返回新令牌
	// 包括令牌字符串和过期时间戳
	return c.JSON(fiber.Map{
		"message":    "刷新令牌成功",
		"token":      newToken,
		"expires_at": expireTime.Unix(),
	})
}

// SalespersonLogout 销售员登出
// 该处理函数用于使当前会话的令牌失效
// 处理流程:
//  1. 从请求头提取当前令牌
//  2. 从数据库中删除该令牌记录
func SalespersonLogout(c *fiber.Ctx) error {
	// 从请求头获取令牌
	// 验证Authorization头是否存在且格式正确
	authHeader := c.Get("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "未提供有效的认证令牌",
		})
	}

	tokenString := authHeader[7:]

	// 将令牌从数据库中删除
	// 使令牌立即失效，防止后续使用
	if err := database.GetDB().Where("token = ?", tokenString).Delete(&models.SalespersonToken{}).Error; err != nil {
		log.Printf("删除令牌失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "登出失败，请稍后重试",
		})
	}

	return c.JSON(fiber.Map{
		"message": "登出成功",
	})
}

// GetLoginDevices 获取登录设备列表
// 该处理函数返回当前销售员的所有活跃登录设备
// 处理流程:
//  1. 获取当前销售员ID
//  2. 查询该销售员的所有未过期令牌
//  3. 构建并返回设备列表
func GetLoginDevices(c *fiber.Ctx) error {
	// 获取当前销售员ID
	// 从请求头中提取经过身份验证的销售员ID
	salespersonID, err := strconv.Atoi(c.Get("X-Salesperson-ID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 查询该销售员的所有有效令牌
	// 只返回未过期的令牌，代表当前活跃的登录会话
	var tokens []models.SalespersonToken
	if err := database.GetDB().Where("salesperson_id = ? AND expired_at > ?", salespersonID, time.Now()).Find(&tokens).Error; err != nil {
		log.Printf("查询登录设备失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询登录设备失败，请稍后重试",
		})
	}

	// 构建设备列表
	// 转换为前端友好的格式，包含必要的设备信息
	devices := make([]fiber.Map, 0, len(tokens))
	for _, token := range tokens {
		devices = append(devices, fiber.Map{
			"id":         token.ID,
			"user_agent": token.UserAgent,
			"ip":         token.IP,
			"created_at": token.CreatedAt,
			"expired_at": token.ExpiredAt,
		})
	}

	return c.JSON(fiber.Map{
		"devices": devices,
	})
}

// LogoutDevice 登出特定设备
// 该处理函数用于使特定设备的会话令牌失效
// 处理流程:
//  1. 获取当前销售员ID
//  2. 获取目标设备ID
//  3. 删除对应的令牌记录
func LogoutDevice(c *fiber.Ctx) error {
	// 获取当前销售员ID
	// 确保只能操作自己的设备
	salespersonID, err := strconv.Atoi(c.Get("X-Salesperson-ID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 获取设备ID
	// 从URL参数中提取目标设备ID
	deviceID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的设备ID",
		})
	}

	// 删除特定设备的令牌
	// 确保只删除属于当前销售员的设备令牌
	result := database.GetDB().Where("id = ? AND salesperson_id = ?", deviceID, salespersonID).Delete(&models.SalespersonToken{})
	if result.Error != nil {
		log.Printf("登出设备失败: %v", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "登出设备失败，请稍后重试",
		})
	}

	// 检查是否找到并删除了记录
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "设备不存在或不属于当前销售员",
		})
	}

	return c.JSON(fiber.Map{
		"message": "设备登出成功",
	})
}

// ForceLogoutSalesperson 强制销售员登出（使所有令牌失效）
// 该处理函数用于管理员强制使特定销售员的所有会话失效
// 处理流程:
//  1. 获取目标销售员ID
//  2. 删除该销售员的所有令牌记录
func ForceLogoutSalesperson(c *fiber.Ctx) error {
	// 获取销售员ID
	// 从URL参数中提取目标销售员ID
	salespersonID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 删除该销售员的所有令牌
	// 使所有设备上的会话立即失效，强制用户重新登录
	if err := database.GetDB().Where("salesperson_id = ?", salespersonID).Delete(&models.SalespersonToken{}).Error; err != nil {
		log.Printf("删除销售员令牌失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "强制登出失败，请稍后重试",
		})
	}

	return c.JSON(fiber.Map{
		"message": "强制登出成功",
	})
}
