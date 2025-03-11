package handlers

import (
	"errors"
	"fmt"
	"go_creation/database"
	"go_creation/models"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"encoding/base32"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var (
	counter     int64
	counterLock sync.Mutex
)

// BatchCreateKeys 批量生成卡密
// 根据指定的卡密类型和数量，批量生成卡密并保存到数据库
func BatchCreateKeys(c *fiber.Ctx) error {
	// 解析请求参数
	type BatchCreateRequest struct {
		TypeID        uint   `json:"type_id"`        // 卡密类型ID
		SoftwareID    uint   `json:"software_id"`    // 软件ID
		Count         int    `json:"count"`          // 生成数量
		CreatorID     uint   `json:"creator_id"`     // 创建者ID
		CreatorType   string `json:"creator_type"`   // 创建者类型：admin或salesperson
		SalespersonID uint   `json:"salesperson_id"` // 销售员ID，当CreatorType为salesperson时使用
	}

	var req BatchCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败",
		})
	}

	// 验证参数
	if req.Count <= 0 || req.Count > 1000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "生成数量必须在1-1000之间",
		})
	}

	// 设置默认值
	if req.CreatorType == "" {
		req.CreatorType = "admin" // 默认为管理员创建
	}

	// 验证创建者类型
	if req.CreatorType != "admin" && req.CreatorType != "salesperson" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的创建者类型，必须为admin或salesperson",
		})
	}

	// 如果是销售员创建，验证销售员ID
	if req.CreatorType == "salesperson" && req.SalespersonID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "销售员ID不能为空",
		})
	}

	// 验证卡密类型是否存在
	var keyType models.KeyType
	if err := database.GetDB().Where("id = ?", req.TypeID).First(&keyType).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的卡密类型",
		})
	}

	// 检查卡密类型状态
	if keyType.Status != "active" || !keyType.IsActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "卡密类型未激活",
		})
	}

	// 验证软件是否存在
	var software models.Software
	if err := database.GetDB().Where("id = ?", req.SoftwareID).First(&software).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的软件ID",
		})
	}

	// 检查软件状态
	if software.Status != "active" || !software.IsActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "软件未激活",
		})
	}

	// 验证卡密类型是否绑定到指定软件
	var binding models.SoftwareKeyType
	if err := database.GetDB().Where("software_id = ? AND key_type_id = ?", req.SoftwareID, req.TypeID).First(&binding).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "该卡密类型未绑定到指定软件",
		})
	}

	// 如果是销售员创建，验证销售员是否有权限
	if req.CreatorType == "salesperson" {
		var salespersonProduct models.SalespersonProduct
		if err := database.GetDB().Where("salesperson_id = ? AND software_id = ? AND key_type_id = ? AND is_active = true",
			req.SalespersonID, req.SoftwareID, req.TypeID).First(&salespersonProduct).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "销售员无权生成该产品的卡密",
				})
			}
			fmt.Printf("查询销售员产品权限失败: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "查询销售员产品权限失败",
			})
		}

		// 检查生成数量限制
		if salespersonProduct.KeyGenLimit > 0 {
			if salespersonProduct.KeysGenerated+req.Count > salespersonProduct.KeyGenLimit {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": fmt.Sprintf("超出卡密生成限制，当前已生成 %d 个，限制 %d 个",
						salespersonProduct.KeysGenerated, salespersonProduct.KeyGenLimit),
				})
			}
		}
	}

	// 生成卡密
	keys := make([]models.Key, req.Count)
	for i := 0; i < req.Count; i++ {
		keys[i] = models.Key{
			TypeID:        req.TypeID,
			TypeName:      keyType.Name,
			SoftwareID:    req.SoftwareID,
			SoftwareName:  software.Name,
			Code:          generateUniqueCode(),    // 生成唯一的卡密码
			KeyCode:       generateUniqueKeyCode(), // 生成唯一的激活码
			Hours:         keyType.Hours,           // 使用卡密类型的有效期
			Price:         keyType.Price,           // 使用卡密类型的价格
			Status:        "unused",                // 初始状态为未使用
			CreatorID:     req.CreatorID,           // 设置创建者ID
			CreatorType:   req.CreatorType,         // 设置创建者类型
			SalespersonID: req.SalespersonID,       // 设置销售员ID
		}
	}

	// 批量保存到数据库，使用事务确保数据一致性
	tx := database.GetDB().Begin()
	if err := tx.Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "开始事务失败",
		})
	}

	// 打印SQL查询语句
	stmt := tx.Session(&gorm.Session{DryRun: true}).Create(&keys).Statement
	sql := stmt.SQL.String()
	fmt.Printf("批量生成卡密 - SQL查询: %s, 参数: %v\n", sql, stmt.Vars)

	if err := tx.Create(&keys).Error; err != nil {
		tx.Rollback() // 发生错误时回滚事务
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "保存卡密失败: " + err.Error(),
		})
	}

	// 如果是销售员创建，更新销售员产品的已生成卡密数量
	if req.CreatorType == "salesperson" {
		var salespersonProduct models.SalespersonProduct
		if err := tx.Where("salesperson_id = ? AND software_id = ? AND key_type_id = ?",
			req.SalespersonID, req.SoftwareID, req.TypeID).First(&salespersonProduct).Error; err == nil {

			// 更新已生成卡密数量
			if err := tx.Model(&models.SalespersonProduct{}).Where("id = ?", salespersonProduct.ID).
				UpdateColumn("keys_generated", gorm.Expr("keys_generated + ?", req.Count)).Error; err != nil {
				tx.Rollback()
				fmt.Printf("更新销售员产品已生成卡密数量失败: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "更新销售员产品已生成卡密数量失败",
				})
			}

			// 创建销售记录
			totalAmount := float64(req.Count) * keyType.Price
			commission := totalAmount * salespersonProduct.CommissionRate

			sale := models.SalespersonSale{
				SalespersonID:  req.SalespersonID,
				KeyID:          0, // 批量生成时不关联具体卡密
				SoftwareID:     req.SoftwareID,
				KeyTypeID:      req.TypeID,
				SaleAmount:     totalAmount,
				CommissionRate: salespersonProduct.CommissionRate,
				Commission:     commission,
				Status:         "pending",
				Notes:          "通过API批量生成",
			}

			// 打印SQL查询语句
			stmt = tx.Session(&gorm.Session{DryRun: true}).Create(&sale).Statement
			sql = stmt.SQL.String()
			fmt.Printf("创建销售记录 - SQL查询: %s, 参数: %v\n", sql, stmt.Vars)

			if err := tx.Create(&sale).Error; err != nil {
				tx.Rollback()
				fmt.Printf("创建销售记录失败: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "创建销售记录失败: " + err.Error(),
				})
			}

			// 更新销售员的总销售额和总佣金
			if err := tx.Model(&models.Salesperson{}).Where("id = ?", req.SalespersonID).Updates(map[string]interface{}{
				"total_sales":      gorm.Expr("total_sales + ?", totalAmount),
				"total_commission": gorm.Expr("total_commission + ?", commission),
			}).Error; err != nil {
				tx.Rollback()
				fmt.Printf("更新销售员销售统计失败: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "更新销售员销售统计失败",
				})
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "提交事务失败",
		})
	}

	// 返回成功响应和生成的卡密列表
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "卡密生成成功",
		"data":    keys,
	})
}

// GetKeys 查询卡密列表
// 根据查询条件获取卡密列表，支持分页和多条件筛选

// ActivateKey 激活卡密
// 根据卡密码和激活码，激活卡密并返回激活结果
func ActivateKey(c *fiber.Ctx) error {
	// 解析请求参数
	type ActivateRequest struct {
		Code        string `json:"code"`         // 卡密码
		KeyCode     string `json:"key_code"`     // 激活码
		SoftwareID  uint   `json:"software_id"`  // 软件ID
		DeviceInfo  string `json:"device_info"`  // 设备信息
		ActivatorID uint   `json:"activator_id"` // 激活者ID
	}

	var req ActivateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败",
		})
	}

	// 验证参数
	if req.Code == "" || req.KeyCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "卡密码和激活码不能为空",
		})
	}

	if req.SoftwareID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "软件ID不能为空",
		})
	}

	// 查询卡密
	var key models.Key
	if err := database.GetDB().Where("code = ? AND key_code = ?", req.Code, req.KeyCode).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "卡密不存在或激活码错误",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询卡密失败",
		})
	}

	// 验证卡密状态
	if key.Status != "unused" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("卡密状态无效: %s", key.Status),
		})
	}

	// 验证卡密是否属于指定软件
	if key.SoftwareID != req.SoftwareID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "卡密不适用于该软件",
		})
	}

	// 验证软件是否存在且激活
	var software models.Software
	if err := database.GetDB().Where("id = ?", req.SoftwareID).First(&software).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "软件不存在",
		})
	}

	if software.Status != "active" || !software.IsActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "软件未激活",
		})
	}

	// 开始事务
	tx := database.GetDB().Begin()
	if err := tx.Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "开始事务失败",
		})
	}

	// 更新卡密状态
	now := time.Now()
	expiredAt := now.Add(time.Duration(key.Hours) * time.Hour)

	key.Status = "used"
	key.UsedAt = &now
	key.ActivatedAt = &now
	key.ExpiredAt = &expiredAt
	key.DeviceInfo = req.DeviceInfo
	key.UserID = &req.ActivatorID

	// 打印SQL查询语句
	stmt := tx.Session(&gorm.Session{DryRun: true}).Save(&key).Statement
	sql := stmt.SQL.String()
	fmt.Printf("激活卡密 - SQL查询: %s, 参数: %v\n", sql, stmt.Vars)

	if err := tx.Save(&key).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "更新卡密状态失败",
		})
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "提交事务失败",
		})
	}

	// 返回激活结果
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "卡密激活成功",
		"data": fiber.Map{
			"key_id":     key.ID,
			"expired_at": key.ExpiredAt,
			"hours":      key.Hours,
			"software":   software.Name,
		},
	})
}

// VoidKey 作废卡密
// 将指定ID的卡密状态设置为作废
func VoidKey(c *fiber.Ctx) error {
	// 获取卡密ID
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的卡密ID",
		})
	}

	// 查询卡密
	var key models.Key
	if err := database.GetDB().Where("id = ?", id).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "卡密不存在",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询卡密失败",
		})
	}

	// 检查卡密状态
	if key.Status == "void" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "卡密已经是作废状态",
		})
	}

	// 更新卡密状态为作废
	// 打印SQL查询语句
	stmt := database.GetDB().Session(&gorm.Session{DryRun: true}).Model(&key).Update("status", "void").Statement
	sql := stmt.SQL.String()
	fmt.Printf("作废卡密 - SQL查询: %s, 参数: %v\n", sql, stmt.Vars)

	if err := database.GetDB().Model(&key).Update("status", "void").Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "作废卡密失败",
		})
	}

	// 返回成功响应
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "卡密已作废",
		"data":    key,
	})
}

// 添加CSV字段转义函数
func escapeCSVField(field string) string {
	if strings.ContainsAny(field, ",\"\n") {
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(field, "\"", "\"\""))
	}
	return field
}

// ExportKeys 导出卡密
// 根据查询条件导出卡密列表，支持CSV和JSON格式
// @Summary 导出卡密
// @Description 导出卡密列表，支持CSV和JSON格式
// @Tags 卡密管理
// @Accept json
// @Produce json,csv
// @Param format query string false "导出格式，支持json和csv，默认为csv"
// @Param query query models.KeyQuery false "查询条件"
// @Success 200 {object} fiber.Map "导出成功"
// @Failure 400 {object} fiber.Map "查询参数错误"
// @Failure 401 {object} fiber.Map "未授权"
// @Failure 500 {object} fiber.Map "服务器内部错误"
// @Router /api/keys/export [get]
func ExportKeys(c *fiber.Ctx) error {
	fmt.Println("====================== 开始导出卡密 ======================")
	// 获取当前登录的销售员信息
	fmt.Printf("请求头: %+v\n", c.GetReqHeaders())
	fmt.Printf("认证信息: %+v\n", c.Locals("salesperson_id"))

	salespersonID, ok := c.Locals("salesperson_id").(uint)
	if !ok {
		// 尝试转换其他类型
		fmt.Printf("salesperson_id类型转换失败，尝试其他类型转换\n")
		switch id := c.Locals("salesperson_id").(type) {
		case int:
			fmt.Printf("salesperson_id是int类型: %d\n", id)
			salespersonID = uint(id)
		case float64:
			fmt.Printf("salesperson_id是float64类型: %f\n", id)
			salespersonID = uint(id)
		case int64:
			fmt.Printf("salesperson_id是int64类型: %d\n", id)
			salespersonID = uint(id)
		default:
			fmt.Printf("无法识别的销售员ID类型: %T, 值: %v\n", c.Locals("salesperson_id"), c.Locals("salesperson_id"))
			// 尝试从请求头获取
			salespersonIDStr := c.Get("X-Salesperson-ID")
			if salespersonIDStr != "" {
				fmt.Printf("从请求头获取到销售员ID: %s\n", salespersonIDStr)
				id, err := strconv.Atoi(salespersonIDStr)
				if err == nil {
					salespersonID = uint(id)
				} else {
					fmt.Printf("销售员ID转换失败: %v\n", err)
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"code":  -1,
						"error": "未授权访问，请先登录",
					})
				}
			} else {
				fmt.Printf("未从请求头获取到销售员ID\n")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"code":  -1,
					"error": "未授权访问，请先登录",
				})
			}
		}
	}

	fmt.Printf("导出卡密 - 当前销售员ID: %d\n", salespersonID)

	// 获取导出格式
	format := c.Query("format", "csv")
	fmt.Printf("导出格式: %s\n", format)

	// 构建查询条件
	db := database.GetDB().Model(&models.Key{})
	fmt.Println("已创建数据库查询")

	// 从查询参数中获取筛选条件
	softwareID, _ := strconv.Atoi(c.Query("software_id", "0"))
	status := c.Query("status", "")
	typeID, _ := strconv.Atoi(c.Query("type_id", "0"))
	code := c.Query("code", "")
	keyCode := c.Query("key_code", "")
	querySalespersonID, _ := strconv.Atoi(c.Query("salesperson_id", "0"))
	startTime := c.Query("start_time", "")
	endTime := c.Query("end_time", "")

	// 打印查询参数
	fmt.Printf("导出卡密 - 查询参数: software_id=%d, status=%s, type_id=%d, code=%s, key_code=%s, salesperson_id=%d\n",
		softwareID, status, typeID, code, keyCode, querySalespersonID)

	// 添加筛选条件
	if softwareID > 0 {
		fmt.Printf("添加软件ID筛选条件: %d\n", softwareID)
		db = db.Where("software_id = ?", softwareID)
	}

	if status != "" {
		fmt.Printf("添加状态筛选条件: %s\n", status)
		db = db.Where("status = ?", status)
	}

	if typeID > 0 {
		fmt.Printf("添加类型ID筛选条件: %d\n", typeID)
		db = db.Where("type_id = ?", typeID)
	}

	if code != "" {
		fmt.Printf("添加卡密码筛选条件: %s\n", code)
		db = db.Where("code LIKE ?", "%"+code+"%")
	}

	if keyCode != "" {
		fmt.Printf("添加激活码筛选条件: %s\n", keyCode)
		db = db.Where("key_code LIKE ?", "%"+keyCode+"%")
	}

	// 销售员只能查看自己的卡密
	// 如果前端传入了其他销售员ID，先检查是否有权限查看该销售员的卡密
	if querySalespersonID > 0 && uint(querySalespersonID) != salespersonID {
		fmt.Printf("检查销售员权限 - 请求的销售员ID: %d, 当前销售员ID: %d\n", querySalespersonID, salespersonID)
		// 检查当前销售员是否有权限查看其他销售员的卡密
		// 这里简化处理，假设ID为1的是管理员，有权限查看所有卡密
		if salespersonID == 1 {
			fmt.Printf("当前销售员是管理员，可以查看其他销售员的卡密\n")
			// 管理员，可以查看指定销售员的卡密
			db = db.Where("salesperson_id = ?", querySalespersonID)
		} else {
			fmt.Printf("当前销售员不是管理员，只能查看自己的卡密\n")
			// 非管理员，只能查看自己的卡密
			db = db.Where("salesperson_id = ?", salespersonID)
		}
	} else {
		fmt.Printf("使用当前销售员ID筛选: %d\n", salespersonID)
		// 未指定销售员ID，使用当前销售员ID
		db = db.Where("salesperson_id = ?", salespersonID)
	}

	// 时间范围筛选
	if startTime != "" {
		fmt.Printf("添加开始时间筛选条件: %s\n", startTime)
		db = db.Where("created_at >= ?", startTime)
	}

	if endTime != "" {
		fmt.Printf("添加结束时间筛选条件: %s\n", endTime)
		db = db.Where("created_at <= ?", endTime)
	}

	fmt.Println("开始执行数据库查询...")
	// 执行查询
	var keys []models.Key
	if err := db.Find(&keys).Error; err != nil {
		fmt.Printf("数据库查询失败: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":  -1,
			"error": "数据库查询失败: " + err.Error(),
		})
	}

	// 检查结果
	if len(keys) == 0 {
		fmt.Println("未找到符合条件的卡密")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"code":    0,
			"message": "未找到符合条件的卡密",
			"data":    []models.Key{},
		})
	}

	fmt.Printf("查询成功，找到 %d 条记录\n", len(keys))

	// 根据格式导出
	if format == "json" {
		fmt.Printf("导出JSON格式，共 %d 条记录\n", len(keys))
		return c.JSON(fiber.Map{
			"code":    0,
			"message": "导出成功",
			"data":    keys,
		})
	} else {
		// 导出CSV格式
		fmt.Printf("导出CSV格式，共 %d 条记录\n", len(keys))

		// 设置响应头
		c.Set("Content-Disposition", "attachment; filename=keys.csv")
		c.Set("Content-Type", "text/csv")

		// 构建CSV内容
		var csvContent strings.Builder
		// 添加CSV头
		csvContent.WriteString("ID,卡密码,激活码,类型ID,类型名称,有效期(小时),价格,软件ID,软件名称,状态,创建者ID,创建者类型,销售员ID,使用者ID,使用设备信息,使用时间,过期时间,激活时间,是否黑名单,创建时间,更新时间\n")

		// 添加数据行
		for _, key := range keys {
			// 处理可能为空的时间字段
			usedAt := ""
			if key.UsedAt != nil {
				usedAt = key.UsedAt.Format("2006-01-02 15:04:05")
			}

			expiredAt := ""
			if key.ExpiredAt != nil {
				expiredAt = key.ExpiredAt.Format("2006-01-02 15:04:05")
			}

			activatedAt := ""
			if key.ActivatedAt != nil {
				activatedAt = key.ActivatedAt.Format("2006-01-02 15:04:05")
			}

			// 处理可能为空的用户ID
			userID := ""
			if key.UserID != nil {
				userID = fmt.Sprintf("%d", *key.UserID)
			}

			// 构建CSV行
			row := fmt.Sprintf("%d,%s,%s,%d,%s,%d,%.2f,%d,%s,%s,%d,%s,%d,%s,%s,%s,%s,%s,%t,%s,%s\n",
				key.ID,
				escapeCSVField(key.Code),
				escapeCSVField(key.KeyCode),
				key.TypeID,
				escapeCSVField(key.TypeName),
				key.Hours,
				key.Price,
				key.SoftwareID,
				escapeCSVField(key.SoftwareName),
				escapeCSVField(key.Status),
				key.CreatorID,
				escapeCSVField(key.CreatorType),
				key.SalespersonID,
				userID,
				escapeCSVField(key.DeviceInfo),
				usedAt,
				expiredAt,
				activatedAt,
				key.IsBlacklisted,
				key.CreatedAt.Format("2006-01-02 15:04:05"),
				key.UpdatedAt.Format("2006-01-02 15:04:05"))
			csvContent.WriteString(row)
		}

		fmt.Println("CSV构建完成，准备发送响应")
		return c.SendString(csvContent.String())
	}
}

// GetKeyStatus 获取卡密状态
// 根据卡密ID、卡密码或激活码查询卡密状态
// 支持多种查询方式：
// 1. 通过卡密ID查询
// 2. 通过卡密码查询
// 3. 通过激活码查询
// 4. 通过软件ID查询
func GetKeyStatus(c *fiber.Ctx) error {
	fmt.Println("====================== 开始查询卡密状态 ======================")

	// 获取查询参数
	id, _ := strconv.Atoi(c.Query("id", "0"))
	code := c.Query("code")
	keyCode := c.Query("key_code")
	softwareID, _ := strconv.Atoi(c.Query("software_id", "0"))

	fmt.Printf("查询参数 - ID: %d, Code: %s, KeyCode: %s, SoftwareID: %d\n", id, code, keyCode, softwareID)

	// 验证查询参数
	if id == 0 && code == "" && keyCode == "" && softwareID == 0 {
		fmt.Println("缺少查询参数")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":  -1,
			"error": "请至少提供一个查询条件：id、code、key_code 或 software_id",
		})
	}

	// 构建查询
	db := database.GetDB().Model(&models.Key{})

	// 根据提供的参数构建查询条件
	if id > 0 {
		fmt.Printf("根据ID查询卡密: %d\n", id)
		db = db.Where("id = ?", id)
	}
	if code != "" {
		fmt.Printf("根据卡密码查询卡密: %s\n", code)
		db = db.Where("code = ?", code)
	}
	if keyCode != "" {
		fmt.Printf("根据激活码查询卡密: %s\n", keyCode)
		db = db.Where("key_code = ?", keyCode)
	}
	if softwareID > 0 {
		fmt.Printf("根据软件ID查询卡密: %d\n", softwareID)
		db = db.Where("software_id = ?", softwareID)
	}

	// 至少需要提供一个查询条件
	if id == 0 && code == "" && keyCode == "" && softwareID == 0 {
		fmt.Println("缺少查询参数")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":  -1,
			"error": "请至少提供一个查询条件：id、code、key_code 或 software_id",
		})
	}

	// 执行查询
	var keys []models.Key
	if err := db.Find(&keys).Error; err != nil {
		fmt.Println("查询卡密状态 - 数据库查询失败:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":  -1,
			"error": "数据库查询失败: " + err.Error(),
		})
	}

	// 检查结果是否为空
	if len(keys) == 0 {
		fmt.Println("查询卡密状态 - 未找到匹配的卡密")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"code":  -1,
			"error": "未找到匹配的卡密",
		})
	}

	fmt.Printf("查询成功，找到 %d 条记录\n", len(keys))

	// 返回查询结果
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "查询成功",
		"data":    keys,
	})
}

// GetAllKeys 获取所有卡密
// 支持分页、按状态筛选、按类型筛选、按软件筛选等功能
func GetAllKeys(c *fiber.Ctx) error {
	// 解析查询参数
	var query models.KeyQuery
	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":  -1,
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
	// 限制最大页面大小
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	// 构建查询条件
	db := database.GetDB().Model(&models.Key{})

	// 按状态筛选
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
		fmt.Printf("按状态筛选: %s\n", query.Status)
	}

	// 按卡密类型筛选
	if query.TypeID > 0 {
		db = db.Where("type_id = ?", query.TypeID)
		fmt.Printf("按类型ID筛选: %d\n", query.TypeID)
	}

	// 按软件ID筛选
	if query.SoftwareID > 0 {
		db = db.Where("software_id = ?", query.SoftwareID)
		fmt.Printf("按软件ID筛选: %d\n", query.SoftwareID)
	}

	// 按创建者筛选
	if query.CreatorID > 0 {
		db = db.Where("creator_id = ?", query.CreatorID)
		fmt.Printf("按创建者ID筛选: %d\n", query.CreatorID)
	}

	// 按激活者筛选
	if query.ActivatorID > 0 {
		db = db.Where("activator_id = ?", query.ActivatorID)
		fmt.Printf("按激活者ID筛选: %d\n", query.ActivatorID)
	}

	// 按创建时间范围筛选
	if query.StartTime != "" {
		db = db.Where("created_at >= ?", query.StartTime)
		fmt.Printf("按开始时间筛选: %s\n", query.StartTime)
	}
	if query.EndTime != "" {
		db = db.Where("created_at <= ?", query.EndTime)
		fmt.Printf("按结束时间筛选: %s\n", query.EndTime)
	}

	// 按销售员ID筛选
	if query.SalespersonID > 0 {
		db = db.Where("salesperson_id = ?", query.SalespersonID)
		fmt.Printf("按销售员ID筛选: %d\n", query.SalespersonID)
	}

	// 打印SQL查询语句
	stmt := db.Session(&gorm.Session{DryRun: true}).Find(&models.Key{}).Statement
	sql := stmt.SQL.String()
	fmt.Printf("SQL查询: %s, 参数: %v\n", sql, stmt.Vars)

	// 计算总记录数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":  -1,
			"error": "查询卡密总数失败",
		})
	}

	// 计算分页偏移量
	offset := (query.Page - 1) * query.PageSize

	// 查询分页数据
	var keys []models.Key
	if err := db.Offset(offset).Limit(query.PageSize).Order("id DESC").Find(&keys).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":  -1,
			"error": "查询卡密列表失败",
		})
	}

	// 返回分页结果
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "查询成功",
		"data": fiber.Map{
			"list":      keys,
			"total":     total,
			"page":      query.Page,
			"page_size": query.PageSize,
			"pages":     int(math.Ceil(float64(total) / float64(query.PageSize))),
		},
	})
}

// GetKeysBySoftwareID 按软件ID查询卡密
// 获取指定软件的所有卡密，支持分页和状态筛选
func GetKeysBySoftwareID(c *fiber.Ctx) error {
	// 获取软件ID
	softwareID, err := c.ParamsInt("id")
	if err != nil || softwareID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":  -1,
			"error": "无效的软件ID",
		})
	}

	// 解析查询参数
	var query models.KeyQuery
	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":  -1,
			"error": "查询参数解析失败",
		})
	}

	// 设置软件ID
	query.SoftwareID = uint(softwareID)

	// 设置默认分页参数
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}
	// 限制最大页面大小
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	// 构建查询条件
	db := database.GetDB().Model(&models.Key{}).Where("software_id = ?", softwareID)

	// 按状态筛选
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
		fmt.Printf("按状态筛选: %s\n", query.Status)
	}

	// 按卡密类型筛选
	if query.TypeID > 0 {
		db = db.Where("type_id = ?", query.TypeID)
		fmt.Printf("按类型ID筛选: %d\n", query.TypeID)
	}

	// 按创建者筛选
	if query.CreatorID > 0 {
		db = db.Where("creator_id = ?", query.CreatorID)
		fmt.Printf("按创建者ID筛选: %d\n", query.CreatorID)
	}

	// 按激活者筛选
	if query.ActivatorID > 0 {
		db = db.Where("activator_id = ?", query.ActivatorID)
		fmt.Printf("按激活者ID筛选: %d\n", query.ActivatorID)
	}

	// 按创建时间范围筛选
	if query.StartTime != "" {
		db = db.Where("created_at >= ?", query.StartTime)
		fmt.Printf("按开始时间筛选: %s\n", query.StartTime)
	}
	if query.EndTime != "" {
		db = db.Where("created_at <= ?", query.EndTime)
		fmt.Printf("按结束时间筛选: %s\n", query.EndTime)
	}

	// 按销售员ID筛选
	if query.SalespersonID > 0 {
		db = db.Where("salesperson_id = ?", query.SalespersonID)
		fmt.Printf("按销售员ID筛选: %d\n", query.SalespersonID)
	}

	// 打印SQL查询语句
	stmt := db.Session(&gorm.Session{DryRun: true}).Find(&models.Key{}).Statement
	sql := stmt.SQL.String()
	fmt.Printf("SQL查询: %s, 参数: %v\n", sql, stmt.Vars)

	// 计算总记录数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":  -1,
			"error": "查询卡密总数失败",
		})
	}

	// 计算分页偏移量
	offset := (query.Page - 1) * query.PageSize

	// 查询分页数据
	var keys []models.Key
	if err := db.Offset(offset).Limit(query.PageSize).Order("id DESC").Find(&keys).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":  -1,
			"error": "查询卡密列表失败",
		})
	}

	// 返回分页结果
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "查询成功",
		"data": fiber.Map{
			"list":      keys,
			"total":     total,
			"page":      query.Page,
			"page_size": query.PageSize,
			"pages":     int(math.Ceil(float64(total) / float64(query.PageSize))),
		},
	})
}

// GetKeyByID 获取单个卡密详情
// 根据卡密ID获取卡密的详细信息
func GetKeyByID(c *fiber.Ctx) error {
	// 获取卡密ID
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":  -1,
			"error": "无效的卡密ID",
		})
	}

	// 查询卡密
	var key models.Key
	if err := database.GetDB().Where("id = ?", id).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code":  -1,
				"error": "卡密不存在",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":  -1,
			"error": "查询卡密失败",
		})
	}

	// 打印SQL查询语句
	stmt := database.GetDB().Session(&gorm.Session{DryRun: true}).Where("id = ?", id).First(&models.Key{}).Statement
	sql := stmt.SQL.String()
	fmt.Printf("SQL查询: %s, 参数: %v\n", sql, stmt.Vars)

	// 返回卡密详情
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "查询成功",
		"data":    key,
	})
}

// 生成唯一的卡密码
func generateUniqueCode() string {
	counterLock.Lock()
	static := atomic.AddInt64(&counter, 1)
	counterLock.Unlock()

	// 使用crypto/rand代替math/rand以提高安全性
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// 如果crypto/rand失败，回退到使用时间种子
		seed := time.Now().UnixNano() + static
		r := rand.New(rand.NewSource(seed))
		for i := range bytes {
			bytes[i] = byte(r.Intn(256))
		}
	}

	// 使用base32编码，去除可能混淆的字符
	str := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
	str = strings.ReplaceAll(str, "1", "") // 移除数字1
	str = strings.ReplaceAll(str, "0", "") // 移除数字0
	str = strings.ReplaceAll(str, "O", "") // 移除字母O
	str = strings.ReplaceAll(str, "I", "") // 移除字母I
	str = strings.ReplaceAll(str, "L", "") // 移除字母L

	if len(str) < 16 {
		// 如果长度不够，补充随机字符
		charset := "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
		for len(str) < 16 {
			pos := rand.Intn(len(charset))
			str += string(charset[pos])
		}
	}

	// 截取16位并格式化
	str = str[:16]
	return fmt.Sprintf("%s-%s-%s-%s",
		str[0:4], str[4:8], str[8:12], str[12:16])
}

// 生成唯一的激活码
func generateUniqueKeyCode() string {
	counterLock.Lock()
	static := atomic.AddInt64(&counter, 1)
	counterLock.Unlock()

	// 使用crypto/rand代替math/rand
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		// 如果crypto/rand失败，回退到使用时间种子
		seed := time.Now().UnixNano() + static + 1000000
		r := rand.New(rand.NewSource(seed))
		for i := range bytes {
			bytes[i] = byte(r.Intn(256))
		}
	}

	// 使用base32编码，去除可能混淆的字符
	str := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
	str = strings.ReplaceAll(str, "1", "") // 移除数字1
	str = strings.ReplaceAll(str, "0", "") // 移除数字0
	str = strings.ReplaceAll(str, "O", "") // 移除字母O
	str = strings.ReplaceAll(str, "I", "") // 移除字母I
	str = strings.ReplaceAll(str, "L", "") // 移除字母L

	if len(str) < 8 {
		// 如果长度不够，补充随机字符
		charset := "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
		for len(str) < 8 {
			pos := rand.Intn(len(charset))
			str += string(charset[pos])
		}
	}

	// 截取8位
	return str[:8]
}
