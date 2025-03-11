package handlers

import (
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go_creation/database"
	"go_creation/models"
)

// CreateSoftware 创建新软件
// 接收软件的基本信息，创建新的软件记录并保存到数据库
func CreateSoftware(c *fiber.Ctx) error {
	// 解析请求体中的软件数据
	var software models.Software
	if err := c.BodyParser(&software); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败: " + err.Error(),
		})
	}

	// 验证软件名称是否为空
	if software.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "软件名称不能为空",
		})
	}
	
	// 验证版本号是否为空
	if software.Version == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "版本号不能为空",
		})
	}
	
	// 验证描述是否为空
	if software.Description == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "软件描述不能为空",
		})
	}

	// 验证软件名称是否已存在
	var existingSoftware models.Software
	result := database.GetDB().Where("name = ?", software.Name).First(&existingSoftware)
	if result.Error == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "软件名称已存在",
		})
	} else if result.Error != gorm.ErrRecordNotFound {
		// 如果发生其他错误，返回服务器错误
		log.Printf("查询软件失败: %v", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询软件失败",
		})
	}

	// 设置默认值
	if software.Status == "" {
		software.Status = "active" // 默认状态为活跃
	}
	software.IsActive = true // 默认启用
	
	// 设置创建时间
	software.CreatedAt = time.Now()
	software.UpdatedAt = time.Now()

	// 保存软件到数据库
	if err := database.GetDB().Create(&software).Error; err != nil {
		log.Printf("创建软件失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "创建软件失败: " + err.Error(),
		})
	}

	// 返回成功响应和创建的软件数据
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "软件创建成功",
		"data":    software,
	})
}

// GetAllSoftware 获取所有软件
// 支持分页和筛选，返回软件列表
func GetAllSoftware(c *fiber.Ctx) error {
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
	query := database.GetDB().Model(&models.Software{})

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
		log.Printf("计算软件总数失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询软件总数失败",
		})
	}

	// 分页
	offset := (page - 1) * limit
	query = query.Offset(offset).Limit(limit)

	// 排序
	query = query.Order("created_at DESC")

	// 执行查询
	var softwares []models.Software
	if err := query.Find(&softwares).Error; err != nil {
		log.Printf("查询软件列表失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询软件列表失败",
		})
	}

	// 返回结果
	return c.JSON(fiber.Map{
		"data": softwares,
		"meta": fiber.Map{
			"total":  total,
			"page":   page,
			"limit":  limit,
			"pages":  (total + int64(limit) - 1) / int64(limit),
			"offset": offset,
		},
	})
}

// GetSoftwareByID 根据ID获取软件
// 返回指定ID的软件详细信息
func GetSoftwareByID(c *fiber.Ctx) error {
	// 获取路径参数中的ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的ID参数",
		})
	}

	// 查询软件
	var software models.Software
	if err := database.GetDB().First(&software, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "软件不存在",
			})
		}
		log.Printf("查询软件失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询软件失败",
		})
	}

	// 返回软件数据
	return c.JSON(fiber.Map{
		"data": software,
	})
}

// UpdateSoftware 更新软件
func UpdateSoftware(c *fiber.Ctx) error {
	// 获取软件ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "无效的软件ID",
		})
	}

	// 查询软件是否存在
	var software models.Software
	if err := database.GetDB().First(&software, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "软件不存在",
		})
	}

	// 解析请求参数
	var updateData models.Software
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "参数解析失败: " + err.Error(),
		})
	}

	// 更新软件
	if err := database.GetDB().Model(&software).Updates(updateData).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "更新软件失败: " + err.Error(),
		})
	}

	// 重新查询最新数据
	if err := database.GetDB().First(&software, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "查询更新后的软件失败",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "软件更新成功",
		"data":    software,
	})
}

// DeleteSoftware 删除软件
func DeleteSoftware(c *fiber.Ctx) error {
	// 获取软件ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "无效的软件ID",
		})
	}

	// 查询软件是否存在
	var software models.Software
	if err := database.GetDB().First(&software, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "软件不存在",
		})
	}

	// 删除软件
	if err := database.GetDB().Delete(&software).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "删除软件失败: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "软件删除成功",
	})
}

// ActivateSoftware 激活软件
func ActivateSoftware(c *fiber.Ctx) error {
	// 获取软件ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "无效的软件ID",
		})
	}

	// 查询软件是否存在
	var software models.Software
	if err := database.GetDB().First(&software, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "软件不存在",
		})
	}

	// 更新软件状态
	updates := map[string]interface{}{
		"status":    "active",
		"is_active": true,
	}

	if err := database.GetDB().Model(&software).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "激活软件失败: " + err.Error(),
		})
	}

	// 重新查询最新数据
	if err := database.GetDB().First(&software, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "查询更新后的软件失败",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "软件激活成功",
		"data":    software,
	})
}

// DeactivateSoftware 停用软件
func DeactivateSoftware(c *fiber.Ctx) error {
	// 获取软件ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "无效的软件ID",
		})
	}

	// 查询软件是否存在
	var software models.Software
	if err := database.GetDB().First(&software, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "软件不存在",
		})
	}

	// 更新软件状态
	updates := map[string]interface{}{
		"status":    "inactive",
		"is_active": false,
	}

	if err := database.GetDB().Model(&software).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "停用软件失败: " + err.Error(),
		})
	}

	// 重新查询最新数据
	if err := database.GetDB().First(&software, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "查询更新后的软件失败",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "软件停用成功",
		"data":    software,
	})
}

// BindKeyType 将卡密类型绑定到软件
func BindKeyType(c *fiber.Ctx) error {
	// 解析请求参数
	type BindRequest struct {
		SoftwareID uint `json:"software_id"`
		KeyTypeID  uint `json:"key_type_id"`
		CreatorID  uint `json:"creator_id"`
	}

	var req BindRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败",
		})
	}

	// 验证软件是否存在
	var software models.Software
	if err := database.GetDB().First(&software, req.SoftwareID).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "软件不存在",
		})
	}

	// 验证卡密类型是否存在
	var keyType models.KeyType
	if err := database.GetDB().First(&keyType, req.KeyTypeID).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "卡密类型不存在",
		})
	}

	// 检查是否已经绑定
	var existingBinding models.SoftwareKeyType
	result := database.GetDB().Where("software_id = ? AND key_type_id = ?", req.SoftwareID, req.KeyTypeID).First(&existingBinding)
	if result.Error == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "该卡密类型已经绑定到此软件",
		})
	}

	// 创建绑定关系
	binding := models.SoftwareKeyType{
		SoftwareID: req.SoftwareID,
		KeyTypeID:  req.KeyTypeID,
		IsActive:   true,
		CreatorID:  req.CreatorID,
	}

	if err := database.GetDB().Create(&binding).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "绑定卡密类型失败",
		})
	}

	return c.JSON(fiber.Map{
		"message": "卡密类型绑定成功",
		"data":    binding,
	})
}

// UnbindKeyType 解绑卡密类型
func UnbindKeyType(c *fiber.Ctx) error {
	// 解析请求参数
	type UnbindRequest struct {
		SoftwareID uint `json:"software_id"` // 软件ID
		KeyTypeID  uint `json:"key_type_id"` // 卡密类型ID
	}

	var req UnbindRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "参数解析失败: " + err.Error(),
		})
	}

	// 删除绑定关系
	result := database.GetDB().Where("software_id = ? AND key_type_id = ?", req.SoftwareID, req.KeyTypeID).Delete(&models.SoftwareKeyType{})

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "解绑卡密类型失败: " + result.Error.Error(),
		})
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "绑定关系不存在",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "卡密类型解绑成功",
	})
}

// GetSoftwareKeyTypes 获取软件绑定的卡密类型
func GetSoftwareKeyTypes(c *fiber.Ctx) error {
	// 获取软件ID
	softwareID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "无效的软件ID",
		})
	}

	// 查询软件是否存在
	var software models.Software
	if err := database.GetDB().First(&software, softwareID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "软件不存在",
		})
	}

	// 查询绑定的卡密类型
	var bindings []models.SoftwareKeyType
	if err := database.GetDB().Where("software_id = ?", softwareID).Find(&bindings).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "查询绑定关系失败: " + err.Error(),
		})
	}

	// 如果没有绑定关系，返回空数组
	if len(bindings) == 0 {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    []interface{}{},
		})
	}

	// 提取卡密类型ID
	var keyTypeIDs []uint
	for _, binding := range bindings {
		keyTypeIDs = append(keyTypeIDs, binding.KeyTypeID)
	}

	// 查询卡密类型详情
	var keyTypes []models.KeyType
	if err := database.GetDB().Where("id IN ?", keyTypeIDs).Find(&keyTypes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "查询卡密类型详情失败: " + err.Error(),
		})
	}

	// 构建结果
	type KeyTypeWithBinding struct {
		models.KeyType
		IsDefault bool `json:"is_default"`
	}

	var result []KeyTypeWithBinding
	for _, keyType := range keyTypes {
		// 查找对应的绑定关系
		var isDefault bool
		for _, binding := range bindings {
			if binding.KeyTypeID == keyType.ID {
				isDefault = binding.IsActive
				break
			}
		}

		result = append(result, KeyTypeWithBinding{
			KeyType:   keyType,
			IsDefault: isDefault,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}
