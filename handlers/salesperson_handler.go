package handlers

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go_creation/database"
	"go_creation/models"
	"go_creation/utils"
)

// CreateSalesperson 创建新销售员
// 接收销售员的基本信息，创建新的销售员记录并保存到数据库
func CreateSalesperson(c *fiber.Ctx) error {
	// 解析请求体中的销售员数据
	var requestData struct {
		models.Salesperson
		Password string `json:"password"`
	}

	var err error
	if err = c.BodyParser(&requestData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败: " + err.Error(),
		})
	}

	// 将解析的数据复制到销售员对象
	salesperson := requestData.Salesperson

	// 验证必填字段
	if salesperson.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "用户名不能为空",
		})
	}

	if salesperson.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "姓名不能为空",
		})
	}

	// 验证用户名是否已存在
	var existingSalesperson models.Salesperson
	result := database.GetDB().Where("username = ?", salesperson.Username).First(&existingSalesperson)
	if result.Error == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "用户名已存在",
		})
	} else if result.Error != gorm.ErrRecordNotFound {
		// 如果发生其他错误，返回服务器错误
		log.Printf("查询销售员失败: %v", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败",
		})
	}

	// 设置默认状态
	if salesperson.Status == "" {
		salesperson.Status = "active" // 默认状态为在职
	}

	// 处理密码
	if requestData.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "密码不能为空",
		})
	}

	// 设置加密密码
	if err := salesperson.SetPassword(requestData.Password); err != nil {
		log.Printf("密码加密失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "密码加密失败",
		})
	}

	// 保存销售员记录
	if err := database.GetDB().Create(&salesperson).Error; err != nil {
		log.Printf("创建销售员失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "创建销售员失败: " + err.Error(),
		})
	}

	// 返回创建成功的销售员信息
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "销售员创建成功",
		"data":    salesperson,
	})
}

// GetAllSalespersons 获取所有销售员
// 支持分页和筛选
func GetAllSalespersons(c *fiber.Ctx) error {
	// 解析查询参数
	var query models.SalespersonQuery
	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "查询参数解析失败: " + err.Error(),
		})
	}

	// 设置默认分页参数
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	// 构建查询
	db := database.GetDB().Model(&models.Salesperson{})

	// 应用筛选条件
	if query.Username != "" {
		db = db.Where("username LIKE ?", "%"+query.Username+"%")
	}
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.CreatorID != 0 {
		db = db.Where("creator_id = ?", query.CreatorID)
	}

	// 计算总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		log.Printf("计算销售员总数失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "计算销售员总数失败",
		})
	}

	// 获取分页数据
	var salespersons []models.Salesperson
	offset := (query.Page - 1) * query.PageSize
	if err := db.Offset(offset).Limit(query.PageSize).Find(&salespersons).Error; err != nil {
		log.Printf("获取销售员列表失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取销售员列表失败",
		})
	}

	// 返回结果
	return c.JSON(fiber.Map{
		"total": total,
		"page":  query.Page,
		"size":  query.PageSize,
		"data":  salespersons,
	})
}

// GetSalesperson 获取单个销售员信息
func GetSalesperson(c *fiber.Ctx) error {
	// 获取销售员ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 查询销售员
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		log.Printf("获取销售员失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取销售员失败",
		})
	}

	// 返回销售员信息
	return c.JSON(fiber.Map{
		"data": salesperson,
	})
}

// UpdateSalesperson 更新销售员信息
func UpdateSalesperson(c *fiber.Ctx) error {
	// 获取销售员ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 查询销售员是否存在
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		log.Printf("查询销售员失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败",
		})
	}

	// 解析请求体
	var updateData struct {
		Name           string  `json:"name"`
		Phone          string  `json:"phone"`
		Email          string  `json:"email"`
		Status         string  `json:"status"`
		Avatar         string  `json:"avatar"`
		CommissionRate float64 `json:"commission_rate"`
		Password       string  `json:"password"`
	}

	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败: " + err.Error(),
		})
	}

	// 更新字段
	updates := make(map[string]interface{})

	if updateData.Name != "" {
		updates["name"] = updateData.Name
	}
	if updateData.Phone != "" {
		updates["phone"] = updateData.Phone
	}
	if updateData.Email != "" {
		updates["email"] = updateData.Email
	}
	if updateData.Status != "" {
		updates["status"] = updateData.Status
	}
	if updateData.Avatar != "" {
		updates["avatar"] = updateData.Avatar
	}
	if updateData.CommissionRate > 0 {
		updates["commission_rate"] = updateData.CommissionRate
	}

	// 处理密码更新
	if updateData.Password != "" {
		if err := salesperson.SetPassword(updateData.Password); err != nil {
			log.Printf("密码加密失败: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "密码加密失败",
			})
		}
		updates["password"] = salesperson.Password
	}

	// 执行更新
	if err := database.GetDB().Model(&salesperson).Updates(updates).Error; err != nil {
		log.Printf("更新销售员失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "更新销售员失败: " + err.Error(),
		})
	}

	// 重新获取更新后的销售员信息
	if err := database.GetDB().First(&salesperson, id).Error; err != nil {
		log.Printf("获取更新后的销售员信息失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取更新后的销售员信息失败",
		})
	}

	// 返回更新后的销售员信息
	return c.JSON(fiber.Map{
		"message": "销售员信息更新成功",
		"data":    salesperson,
	})
}

// DeleteSalesperson 删除销售员
func DeleteSalesperson(c *fiber.Ctx) error {
	// 获取销售员ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 查询销售员是否存在
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		log.Printf("查询销售员失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败",
		})
	}

	// 执行删除
	if err := database.GetDB().Delete(&salesperson).Error; err != nil {
		log.Printf("删除销售员失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "删除销售员失败: " + err.Error(),
		})
	}

	// 返回成功消息
	return c.JSON(fiber.Map{
		"message": "销售员删除成功",
	})
}

// 处理登录失败响应
func handleLoginFailure(c *fiber.Ctx, username string, message string) error {
	// 记录失败的登录尝试
	isLocked, minutes := utils.DefaultLoginLimiter.RecordFailedLogin(username)

	log.Printf("登录失败，原因: %s, 用户名: %s", message, username)

	var response fiber.Map
	if isLocked {
		response = fiber.Map{
			"error":   "登录尝试次数过多，账号已被临时锁定",
			"minutes": minutes,
		}
	} else {
		remainingAttempts := utils.DefaultLoginLimiter.GetRemainingAttempts(username)
		response = fiber.Map{
			"error":              "用户名或密码错误",
			"remaining_attempts": remainingAttempts,
		}
	}

	return c.Status(fiber.StatusUnauthorized).JSON(response)
}

// SalespersonLogin 销售员登录
func SalespersonLogin(c *fiber.Ctx) error {
	// 解析请求数据
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&loginData); err != nil {
		log.Printf("解析登录数据失败: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败，请检查输入格式",
		})
	}

	// 验证必填字段
	if loginData.Username == "" || loginData.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "用户名和密码不能为空",
		})
	}

	// 检查登录尝试次数限制
	isLocked, remainingMinutes := utils.DefaultLoginLimiter.IsLocked(loginData.Username)
	if isLocked {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":   "登录尝试次数过多，账号已被临时锁定",
			"minutes": remainingMinutes,
		})
	}

	// 查询销售员信息
	var salesperson models.Salesperson
	if err := database.GetDB().Where("username = ?", loginData.Username).First(&salesperson).Error; err != nil {
		// 不要泄露用户是否存在的信息，统一返回用户名或密码错误
		return handleLoginFailure(c, loginData.Username, "用户名不存在")
	}

	// 验证密码
	if !salesperson.CheckPassword(loginData.Password) {
		return handleLoginFailure(c, loginData.Username, "密码错误")
	}

	// 检查销售员状态
	if salesperson.Status != "active" {
		log.Printf("登录失败，账号状态非活跃: %s, 状态 %s", loginData.Username, salesperson.Status)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "账号已被禁用，请联系管理员",
		})
	}

	// 重置登录尝试次数
	utils.DefaultLoginLimiter.ResetAttempts(loginData.Username)

	// 懒惰删除：清理该用户的过期令牌
	if err := database.GetDB().Where("salesperson_id = ? AND expired_at < ?", salesperson.ID, time.Now()).Delete(&models.SalespersonToken{}).Error; err != nil {
		log.Printf("删除过期令牌失败: %v", err)
		// 不返回错误，继续处理
	}

	// 生成JWT令牌，有效期24小时
	token, err := utils.GenerateToken(salesperson.ID, salesperson.Username, 24*time.Hour)
	if err != nil {
		log.Printf("生成令牌失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "登录失败，请稍后重试",
		})
	}

	// 获取客户端信息
	userAgent := c.Get("User-Agent")
	ip := c.IP()

	// 定义过期时间
	expireTime := time.Now().Add(24 * time.Hour)

	// 存储令牌到数据库
	salespersonToken := models.SalespersonToken{
		SalespersonID: salesperson.ID,
		Token:         token,
		UserAgent:     userAgent,
		IP:            ip,
		ExpiredAt:     expireTime,
	}

	if err := database.GetDB().Create(&salespersonToken).Error; err != nil {
		log.Printf("存储令牌失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "登录失败，请稍后重试",
		})
	}

	// 更新最后登录时间
	now := time.Now()
	salesperson.LastLoginAt = &now
	if err := database.GetDB().Model(&salesperson).Update("last_login_at", now).Error; err != nil {
		log.Printf("更新最后登录时间失败: %v", err)
	}

	log.Printf("用户登录成功: %s, ID: %d", salesperson.Username, salesperson.ID)

	// 返回登录成功信息和令牌
	return c.JSON(fiber.Map{
		"message":    "登录成功",
		"token":      token,
		"expires_at": expireTime.Unix(), // 返回过期时间戳，方便前端处理
		"data": fiber.Map{
			"id":       salesperson.ID,
			"username": salesperson.Username,
			"name":     salesperson.Name,
			"status":   salesperson.Status,
			"avatar":   salesperson.Avatar,
		},
	})
}

// AssignProductToSalesperson 为销售员分配产品
func AssignProductToSalesperson(c *fiber.Ctx) error {
	// 解析请求数据
	var assignData struct {
		SalespersonID  uint    `json:"salesperson_id"`
		SoftwareID     uint    `json:"software_id"`
		KeyTypeID      uint    `json:"key_type_id"`
		CommissionRate float64 `json:"commission_rate"`
		KeyGenLimit    int     `json:"key_gen_limit"`
	}

	if err := c.BodyParser(&assignData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败: " + err.Error(),
		})
	}

	// 验证必填字段
	if assignData.SalespersonID == 0 || assignData.SoftwareID == 0 || assignData.KeyTypeID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "销售员ID、软件ID和卡密类型ID不能为空",
		})
	}

	// 验证销售员是否存在
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, assignData.SalespersonID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		log.Printf("查询销售员失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败",
		})
	}

	// 验证软件是否存在
	var software models.Software
	if err := database.GetDB().First(&software, assignData.SoftwareID).Error; err != nil {
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

	// 验证卡密类型是否存在
	var keyType models.KeyType
	if err := database.GetDB().First(&keyType, assignData.KeyTypeID).Error; err != nil {
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

	// 验证软件和卡密类型是否已绑定
	var softwareKeyType models.SoftwareKeyType
	if err := database.GetDB().Where("software_id = ? AND key_type_id = ?", assignData.SoftwareID, assignData.KeyTypeID).First(&softwareKeyType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "软件和卡密类型未绑定，请先绑定",
			})
		}
		log.Printf("查询软件和卡密类型绑定关系失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询软件和卡密类型绑定关系失败",
		})
	}

	// 检查是否已分配
	var existingAssignment models.SalespersonProduct
	result := database.GetDB().Where("salesperson_id = ? AND software_id = ? AND key_type_id = ?",
		assignData.SalespersonID, assignData.SoftwareID, assignData.KeyTypeID).First(&existingAssignment)

	if result.Error == nil {
		// 已存在，更新
		updates := map[string]interface{}{
			"is_active": true,
		}

		if assignData.CommissionRate > 0 {
			updates["commission_rate"] = assignData.CommissionRate
		}

		if assignData.KeyGenLimit > 0 {
			updates["key_gen_limit"] = assignData.KeyGenLimit
		}

		if err := database.GetDB().Model(&existingAssignment).Updates(updates).Error; err != nil {
			log.Printf("更新产品分配失败: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "更新产品分配失败: " + err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"message": "产品分配更新成功",
			"data":    existingAssignment,
		})
	} else if result.Error != gorm.ErrRecordNotFound {
		log.Printf("查询产品分配失败: %v", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询产品分配失败",
		})
	}

	// 创建新的分配
	salespersonProduct := models.SalespersonProduct{
		SalespersonID:  assignData.SalespersonID,
		SoftwareID:     assignData.SoftwareID,
		KeyTypeID:      assignData.KeyTypeID,
		CommissionRate: assignData.CommissionRate,
		KeyGenLimit:    assignData.KeyGenLimit,
		IsActive:       true,
	}

	if err := database.GetDB().Create(&salespersonProduct).Error; err != nil {
		log.Printf("创建产品分配失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "创建产品分配失败: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "产品分配成功",
		"data":    salespersonProduct,
	})
}

// GetSalespersonProducts 获取销售员可销售的产品列表
func GetSalespersonProducts(c *fiber.Ctx) error {
	// 获取销售员ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 查询销售员是否存在
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		log.Printf("查询销售员失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败",
		})
	}

	// 查询销售员可销售的产品
	var products []struct {
		models.SalespersonProduct
		SoftwareName string  `json:"software_name"`
		KeyTypeName  string  `json:"key_type_name"`
		Hours        int     `json:"hours"`
		Price        float64 `json:"price"`
	}

	query := `
		SELECT sp.*, s.name as software_name, kt.name as key_type_name, kt.hours, kt.price
		FROM salesperson_products sp
		JOIN softwares s ON sp.software_id = s.id
		JOIN key_types kt ON sp.key_type_id = kt.id
		WHERE sp.salesperson_id = ? AND sp.is_active = true
	`

	if err := database.GetDB().Raw(query, id).Scan(&products).Error; err != nil {
		log.Printf("查询销售员产品失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员产品失败",
		})
	}

	return c.JSON(fiber.Map{
		"data": products,
	})
}

// GenerateKeysForSalesperson 销售员生成卡密
func GenerateKeysForSalesperson(c *fiber.Ctx) error {
	// 从上下文中获取销售员ID
	salespersonID, ok := c.Locals("salesperson_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "未找到销售员身份信息",
		})
	}

	// 解析请求数据
	var genData struct {
		SoftwareID    uint   `json:"software_id"`
		KeyTypeID     uint   `json:"key_type_id"`
		Count         int    `json:"count"`
		CustomerName  string `json:"customer_name"`
		CustomerPhone string `json:"customer_phone"`
		CustomerEmail string `json:"customer_email"`
		Notes         string `json:"notes"`
	}

	if err := c.BodyParser(&genData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败: " + err.Error(),
		})
	}

	// 验证必填字段
	if genData.SoftwareID == 0 || genData.KeyTypeID == 0 || genData.Count <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "软件ID、卡密类型ID和数量不能为空",
		})
	}

	// 验证销售员是否存在
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, salespersonID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		log.Printf("查询销售员失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败",
		})
	}

	// 验证软件是否存在
	var software models.Software
	if err := database.GetDB().First(&software, genData.SoftwareID).Error; err != nil {
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

	// 验证卡密类型是否存在
	var keyType models.KeyType
	if err := database.GetDB().First(&keyType, genData.KeyTypeID).Error; err != nil {
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

	// 验证软件和卡密类型是否已绑定
	var softwareKeyType models.SoftwareKeyType
	if err := database.GetDB().Where("software_id = ? AND key_type_id = ?", genData.SoftwareID, genData.KeyTypeID).First(&softwareKeyType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "软件和卡密类型未绑定，请先绑定",
			})
		}
		log.Printf("查询软件和卡密类型绑定关系失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询软件和卡密类型绑定关系失败",
		})
	}

	// 检查生成数量限制
	var salespersonProduct models.SalespersonProduct
	if err := database.GetDB().Where("salesperson_id = ? AND software_id = ? AND key_type_id = ? AND is_active = true",
		salespersonID, genData.SoftwareID, genData.KeyTypeID).First(&salespersonProduct).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "销售员无权生成该产品的卡密",
			})
		}
		log.Printf("查询销售员产品权限失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员产品权限失败",
		})
	}

	// 检查生成数量限制
	if salespersonProduct.KeyGenLimit > 0 {
		if salespersonProduct.KeysGenerated+genData.Count > salespersonProduct.KeyGenLimit {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": fmt.Sprintf("超出卡密生成限制，当前已生成 %d 个，限制 %d 个",
					salespersonProduct.KeysGenerated, salespersonProduct.KeyGenLimit),
			})
		}
	}

	// 开始事务
	tx := database.GetDB().Begin()
	if tx.Error != nil {
		log.Printf("开始事务失败: %v", tx.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "开始事务失败",
		})
	}

	// 生成卡密
	keys := make([]models.Key, 0, genData.Count)
	for i := 0; i < genData.Count; i++ {
		// 生成卡密码和激活码
		keyCode := utils.GenerateSalespersonKeyCode()
		code := utils.GenerateSalespersonCode()

		// 创建卡密
		key := models.Key{
			Code:         code,
			KeyCode:      keyCode,
			TypeID:       genData.KeyTypeID,
			TypeName:     keyType.Name,
			Hours:        keyType.Hours,
			Price:        keyType.Price,
			Status:       "unused",
			CreatorID:    salespersonID,
			SoftwareID:   genData.SoftwareID,
			SoftwareName: software.Name,
		}

		if err := tx.Create(&key).Error; err != nil {
			tx.Rollback()
			log.Printf("创建卡密失败: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "创建卡密失败: " + err.Error(),
			})
		}

		keys = append(keys, key)
	}

	// 更新销售员产品的已生成卡密数量
	if err := tx.Model(&models.SalespersonProduct{}).Where("id = ?", salespersonProduct.ID).
		UpdateColumn("keys_generated", gorm.Expr("keys_generated + ?", genData.Count)).Error; err != nil {
		tx.Rollback()
		log.Printf("更新销售员产品已生成卡密数量失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "更新销售员产品已生成卡密数量失败",
		})
	}

	// 创建销售记录
	totalAmount := float64(genData.Count) * keyType.Price
	commission := totalAmount * salespersonProduct.CommissionRate

	sale := models.SalespersonSale{
		SalespersonID:  salespersonID,
		KeyID:          0, // 批量生成时不关联具体卡密
		SoftwareID:     genData.SoftwareID,
		KeyTypeID:      genData.KeyTypeID,
		CustomerName:   genData.CustomerName,
		CustomerPhone:  genData.CustomerPhone,
		CustomerEmail:  genData.CustomerEmail,
		SaleAmount:     totalAmount,
		CommissionRate: salespersonProduct.CommissionRate,
		Commission:     commission,
		Status:         "pending",
		Notes:          genData.Notes,
	}

	if err := tx.Create(&sale).Error; err != nil {
		tx.Rollback()
		log.Printf("创建销售记录失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "创建销售记录失败: " + err.Error(),
		})
	}

	// 更新销售员的总销售额和总佣金
	if err := tx.Model(&models.Salesperson{}).Where("id = ?", salespersonID).
		UpdateColumns(map[string]interface{}{
			"total_sales":      gorm.Expr("total_sales + ?", totalAmount),
			"total_commission": gorm.Expr("total_commission + ?", commission),
		}).Error; err != nil {
		tx.Rollback()
		log.Printf("更新销售员销售统计失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "更新销售员销售统计失败",
		})
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		log.Printf("提交事务失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "提交事务失败",
		})
	}

	// 返回生成的卡密
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "卡密生成成功",
		"data": fiber.Map{
			"keys":       keys,
			"sale":       sale,
			"total":      genData.Count,
			"amount":     totalAmount,
			"commission": commission,
		},
	})
}

// GetSalespersonSales 获取销售员的销售记录
func GetSalespersonSales(c *fiber.Ctx) error {
	// 获取销售员ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 解析查询参数
	var query struct {
		Status    string `query:"status"`
		StartDate string `query:"start_date"`
		EndDate   string `query:"end_date"`
		Page      int    `query:"page"`
		PageSize  int    `query:"page_size"`
	}

	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "查询参数解析失败: " + err.Error(),
		})
	}

	// 设置默认分页参数
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	// 构建查询
	db := database.GetDB().Model(&models.SalespersonSale{}).Where("salesperson_id = ?", id)

	// 按状态筛选
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	// 按时间范围筛选
	if query.StartDate != "" {
		db = db.Where("created_at >= ?", query.StartDate)
	}
	if query.EndDate != "" {
		db = db.Where("created_at <= ?", query.EndDate)
	}

	// 计算总记录数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		log.Printf("计算销售记录总数失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "计算销售记录总数失败",
		})
	}

	// 查询销售记录
	var sales []models.SalespersonSale
	offset := (query.Page - 1) * query.PageSize
	if err := db.Limit(query.PageSize).Offset(offset).Order("created_at DESC").Find(&sales).Error; err != nil {
		log.Printf("获取销售记录失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取销售记录失败",
		})
	}

	// 返回销售记录
	return c.JSON(fiber.Map{
		"total": total,
		"page":  query.Page,
		"size":  query.PageSize,
		"data":  sales,
	})
}

// GetSalespersonCommission 获取销售员的佣金统计
func GetSalespersonCommission(c *fiber.Ctx) error {
	// 获取销售员ID
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 解析查询参数
	var query struct {
		StartDate string `query:"start_date"`
		EndDate   string `query:"end_date"`
	}

	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "查询参数解析失败: " + err.Error(),
		})
	}

	// 构建查询
	db := database.GetDB().Model(&models.SalespersonSale{}).Where("salesperson_id = ?", id)

	// 按时间范围筛选
	if query.StartDate != "" {
		db = db.Where("created_at >= ?", query.StartDate)
	}

	if query.EndDate != "" {
		db = db.Where("created_at <= ?", query.EndDate)
	}

	// 计算总销售额和总佣金
	type CommissionStats struct {
		TotalSales      float64 `json:"total_sales"`
		TotalCommission float64 `json:"total_commission"`
		PendingAmount   float64 `json:"pending_amount"`
		SettledAmount   float64 `json:"settled_amount"`
		CancelledAmount float64 `json:"cancelled_amount"`
	}

	var stats CommissionStats

	// 计算总销售额和总佣金
	if err := db.Select("SUM(sale_amount) as total_sales, SUM(commission) as total_commission").Scan(&stats).Error; err != nil {
		log.Printf("计算佣金统计失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "计算佣金统计失败",
		})
	}

	// 计算待结算金额
	if err := db.Where("status = ?", "pending").Select("SUM(commission) as pending_amount").Scan(&stats).Error; err != nil {
		log.Printf("计算待结算金额失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "计算待结算金额失败",
		})
	}

	// 计算已结算金额
	if err := db.Where("status = ?", "settled").Select("SUM(commission) as settled_amount").Scan(&stats).Error; err != nil {
		log.Printf("计算已结算金额失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "计算已结算金额失败",
		})
	}

	// 计算已取消金额
	if err := db.Where("status = ?", "cancelled").Select("SUM(commission) as cancelled_amount").Scan(&stats).Error; err != nil {
		log.Printf("计算已取消金额失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "计算已取消金额失败",
		})
	}

	// 获取销售员信息
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, id).Error; err != nil {
		log.Printf("获取销售员信息失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "获取销售员信息失败",
		})
	}

	// 返回结果
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"stats": stats,
			"salesperson": fiber.Map{
				"id":               salesperson.ID,
				"name":             salesperson.Name,
				"total_sales":      salesperson.TotalSales,
				"total_commission": salesperson.TotalCommission,
			},
		},
	})
}

// GetSalespersonOwnProducts 获取销售员自己可销售的产品
func GetSalespersonOwnProducts(c *fiber.Ctx) error {
	// 从上下文中获取销售员ID
	salespersonID, ok := c.Locals("salesperson_id").(uint)
	if !ok {
		log.Printf("未找到销售员身份信息: %v", c.Locals("salesperson_id"))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "未找到销售员身份信息",
		})
	}

	// 设置X-Salesperson-Id头，用于调试
	c.Set("X-Salesperson-Id", fmt.Sprintf("%d", salespersonID))

	// 查询销售员可销售的产品
	var products []struct {
		ID             uint    `json:"id"`
		SalespersonID  uint    `json:"salesperson_id"`
		SoftwareID     uint    `json:"software_id"`
		SoftwareName   string  `json:"software_name"`
		KeyTypeID      uint    `json:"key_type_id"`
		KeyTypeName    string  `json:"key_type_name"`
		Hours          int     `json:"hours"`
		Price          float64 `json:"price"`
		CommissionRate float64 `json:"commission_rate"`
		KeyGenLimit    int     `json:"key_gen_limit"`
		KeysGenerated  int     `json:"keys_generated"`
		IsActive       bool    `json:"is_active"`
	}

	// 使用JOIN查询获取完整的产品信息
	query := `
		SELECT 
			sp.id, 
			sp.salesperson_id, 
			sp.software_id, 
			s.name AS software_name, 
			sp.key_type_id, 
			kt.name AS key_type_name, 
			kt.hours, 
			kt.price, 
			sp.commission_rate, 
			sp.key_gen_limit, 
			sp.keys_generated, 
			sp.is_active
		FROM 
			salesperson_products sp
		JOIN 
			softwares s ON sp.software_id = s.id
		JOIN 
			key_types kt ON sp.key_type_id = kt.id
		WHERE 
			sp.salesperson_id = ? AND sp.is_active = true AND s.is_active = true AND kt.is_active = true
		ORDER BY 
			s.name, kt.name
	`

	if err := database.GetDB().Raw(query, salespersonID).Scan(&products).Error; err != nil {
		log.Printf("查询销售员产品失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询产品失败",
		})
	}

	// 如果没有找到产品，返回空数组而不是错误
	if len(products) == 0 {
		return c.JSON(fiber.Map{
			"data": []struct{}{},
		})
	}

	// 返回产品列表
	return c.JSON(fiber.Map{
		"data": products,
	})
}

// GetSalespersonOwnSales 获取销售员自己的销售记录
func GetSalespersonOwnSales(c *fiber.Ctx) error {
	// 从上下文中获取销售员ID
	salespersonID, ok := c.Locals("salesperson_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "未找到销售员身份信息",
		})
	}

	// 解析查询参数
	var query struct {
		Status    string `query:"status"`
		StartDate string `query:"start_date"`
		EndDate   string `query:"end_date"`
		Page      int    `query:"page"`
		PageSize  int    `query:"page_size"`
	}

	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "查询参数解析失败",
		})
	}

	// 设置默认分页参数
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	// 构建查询条件
	db := database.GetDB().Model(&models.SalespersonSale{}).Where("salesperson_id = ?", salespersonID)

	// 按状态筛选
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	// 按时间范围筛选
	if query.StartDate != "" {
		db = db.Where("created_at >= ?", query.StartDate)
	}
	if query.EndDate != "" {
		db = db.Where("created_at <= ?", query.EndDate)
	}

	// 计算总记录数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		log.Printf("计算销售记录总数失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售记录失败",
		})
	}

	// 查询销售记录
	var sales []models.SalespersonSale
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&sales).Error; err != nil {
		log.Printf("查询销售记录失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售记录失败",
		})
	}

	// 返回销售记录
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"list":      sales,
			"total":     total,
			"page":      query.Page,
			"page_size": query.PageSize,
			"pages":     int(math.Ceil(float64(total) / float64(query.PageSize))),
		},
	})
}

// GetSalespersonOwnCommission 获取销售员自己的佣金统计
func GetSalespersonOwnCommission(c *fiber.Ctx) error {
	// 从上下文中获取销售员ID
	salespersonID, ok := c.Locals("salesperson_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "未找到销售员身份信息",
		})
	}

	// 查询销售员信息
	var salesperson models.Salesperson
	if err := database.GetDB().Where("id = ?", salespersonID).First(&salesperson).Error; err != nil {
		log.Printf("查询销售员失败: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员信息失败",
		})
	}

	// 查询佣金统计
	type CommissionStats struct {
		TotalSales      float64 `json:"total_sales"`
		TotalCommission float64 `json:"total_commission"`
		PendingAmount   float64 `json:"pending_amount"`
		SettledAmount   float64 `json:"settled_amount"`
		CancelledAmount float64 `json:"cancelled_amount"`
	}

	var stats CommissionStats

	// 设置总销售额和总佣金
	stats.TotalSales = salesperson.TotalSales
	stats.TotalCommission = salesperson.TotalCommission

	// 查询待结算佣金
	if err := database.GetDB().Model(&models.SalespersonSale{}).
		Where("salesperson_id = ? AND status = ?", salespersonID, "pending").
		Select("COALESCE(SUM(commission), 0) as pending_amount").
		Scan(&stats.PendingAmount).Error; err != nil {
		log.Printf("查询待结算佣金失败: %v", err)
	}

	// 查询已结算佣金
	if err := database.GetDB().Model(&models.SalespersonSale{}).
		Where("salesperson_id = ? AND status = ?", salespersonID, "settled").
		Select("COALESCE(SUM(commission), 0) as settled_amount").
		Scan(&stats.SettledAmount).Error; err != nil {
		log.Printf("查询已结算佣金失败: %v", err)
	}

	// 查询已取消佣金
	if err := database.GetDB().Model(&models.SalespersonSale{}).
		Where("salesperson_id = ? AND status = ?", salespersonID, "cancelled").
		Select("COALESCE(SUM(commission), 0) as cancelled_amount").
		Scan(&stats.CancelledAmount).Error; err != nil {
		log.Printf("查询已取消佣金失败: %v", err)
	}

	// 返回佣金统计
	return c.JSON(fiber.Map{
		"data": stats,
	})
}
