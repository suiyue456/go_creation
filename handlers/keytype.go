package handlers

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go_creation/database"
	"go_creation/models"
)

// CreateKeyType 创建卡密类型
// 接收卡密类型的基本信息，创建新的卡密类型并保存到数据库
func CreateKeyType(c *fiber.Ctx) error {
	// 解析请求体中的卡密类型数据
	var keyType models.KeyType
	if err := c.BodyParser(&keyType); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败: " + err.Error(),
		})
	}

	// 验证卡密类型名称是否为空
	if keyType.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "卡密类型名称不能为空",
		})
	}

	// 验证卡密类型名称是否已存在
	var existingKeyType models.KeyType
	result := database.GetDB().Where("name = ?", keyType.Name).First(&existingKeyType)
	if result.Error == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "卡密类型名称已存在",
		})
	} else if result.Error != gorm.ErrRecordNotFound {
		// 如果发生其他错误，返回服务器错误
		log.Printf("查询卡密类型失败: %v", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询卡密类型失败",
		})
	}

	// 设置默认值
	if keyType.Status == "" {
		keyType.Status = "active" // 默认状态为活跃
	}
	keyType.IsActive = true // 默认启用

	// 保存卡密类型到数据库
	if err := database.GetDB().Create(&keyType).Error; err != nil {
		log.Printf("创建卡密类型失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "创建卡密类型失败: " + err.Error(),
		})
	}

	// 返回成功响应和创建的卡密类型数据
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "卡密类型创建成功",
		"data":    keyType,
	})
}

// GetKeyTypeByID 根据ID获取卡密类型
// 返回指定ID的卡密类型详细信息
func GetKeyTypeByID(c *fiber.Ctx) error {
	// 获取路径参数中的ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的ID参数",
		})
	}

	// 查询卡密类型
	var keyType models.KeyType
	if err := database.GetDB().First(&keyType, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "卡密类型不存在",
			})
		}
		log.Printf("查询卡密类型失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询卡密类型失败",
		})
	}

	// 返回卡密类型数据
	return c.JSON(fiber.Map{
		"data": keyType,
	})
}

// GetAllKeyTypes 获取所有卡密类型
// 支持分页和筛选，返回卡密类型列表
func GetAllKeyTypes(c *fiber.Ctx) error {
	// 解析查询参数
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	name := c.Query("name")
	status := c.Query("status")
	isActive := c.Query("is_active")

	// 验证分页参数
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// 构建查询
	query := database.GetDB().Model(&models.KeyType{})

	// 应用过滤条件
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if isActive != "" {
		active, err := strconv.ParseBool(isActive)
		if err == nil {
			query = query.Where("is_active = ?", active)
		}
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("计算卡密类型总数失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询卡密类型总数失败",
		})
	}

	// 分页
	offset := (page - 1) * limit
	query = query.Offset(offset).Limit(limit)

	// 排序
	query = query.Order("created_at DESC")

	// 执行查询
	var keyTypes []models.KeyType
	if err := query.Find(&keyTypes).Error; err != nil {
		log.Printf("查询卡密类型列表失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询卡密类型列表失败",
		})
	}

	// 返回结果
	return c.JSON(fiber.Map{
		"data": keyTypes,
		"meta": fiber.Map{
			"total":  total,
			"page":   page,
			"limit":  limit,
			"pages":  (total + int64(limit) - 1) / int64(limit),
			"offset": offset,
		},
	})
}

// UpdateKeyType 更新卡密类型
func UpdateKeyType(c *fiber.Ctx) error {
	// 获取路径参数
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "无效的ID: " + err.Error(),
		})
	}

	// 解析请求参数
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "参数解析失败: " + err.Error(),
		})
	}

	// 检查卡密类型是否存在
	var keyType models.KeyType
	if err := database.GetDB().First(&keyType, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "卡密类型不存在",
		})
	}

	// 更新卡密类型
	if err := database.GetDB().Model(&keyType).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "更新卡密类型失败: " + err.Error(),
		})
	}

	// 获取更新后的卡密类型
	if err := database.GetDB().First(&keyType, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "获取更新后的卡密类型失败: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "卡密类型更新成功",
		"data":    keyType,
	})
}

// DeleteKeyType 删除卡密类型
func DeleteKeyType(c *fiber.Ctx) error {
	// 获取路径参数
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "无效的ID: " + err.Error(),
		})
	}

	// 检查卡密类型是否存在
	var keyType models.KeyType
	if err := database.GetDB().First(&keyType, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "卡密类型不存在",
		})
	}

	// 检查是否有关联的卡密
	var count int64
	if err := database.GetDB().Model(&models.Key{}).Where("type_id = ?", id).Count(&count).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "检查关联卡密失败: " + err.Error(),
		})
	}

	if count > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "该卡密类型下存在卡密，无法删除",
		})
	}

	// 删除卡密类型
	if err := database.GetDB().Delete(&keyType).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "删除卡密类型失败: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "卡密类型删除成功",
	})
}

// ActivateKeyType 激活卡密类型
func ActivateKeyType(c *fiber.Ctx) error {
	// 获取路径参数
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "无效的ID: " + err.Error(),
		})
	}

	// 检查卡密类型是否存在
	var keyType models.KeyType
	if err := database.GetDB().First(&keyType, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "卡密类型不存在",
		})
	}

	// 激活卡密类型
	updates := map[string]interface{}{
		"is_active": true,
		"status":    "active",
	}
	if err := database.GetDB().Model(&keyType).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "激活卡密类型失败: " + err.Error(),
		})
	}

	// 获取更新后的卡密类型
	if err := database.GetDB().First(&keyType, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "获取更新后的卡密类型失败: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "卡密类型激活成功",
		"data":    keyType,
	})
}

// DeactivateKeyType 停用卡密类型
func DeactivateKeyType(c *fiber.Ctx) error {
	// 获取路径参数
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "无效的ID: " + err.Error(),
		})
	}

	// 检查卡密类型是否存在
	var keyType models.KeyType
	if err := database.GetDB().First(&keyType, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "卡密类型不存在",
		})
	}

	// 停用卡密类型
	updates := map[string]interface{}{
		"is_active": false,
		"status":    "inactive",
	}
	if err := database.GetDB().Model(&keyType).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "停用卡密类型失败: " + err.Error(),
		})
	}

	// 获取更新后的卡密类型
	if err := database.GetDB().First(&keyType, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "获取更新后的卡密类型失败: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "卡密类型停用成功",
		"data":    keyType,
	})
}
