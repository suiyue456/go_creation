package handlers

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go_creation/database"
	"go_creation/models"
	"go_creation/utils"
)

// 最大允许的代理层级
const MaxAgentLevel = 5

// GenerateAgentCode 为销售员生成代理码
func GenerateAgentCode(c *fiber.Ctx) error {
	// 获取当前销售员ID
	salespersonID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 查询销售员信息
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, salespersonID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败: " + err.Error(),
		})
	}

	// 如果已有代理码，直接返回
	if salesperson.AgentCode != "" {
		return c.JSON(fiber.Map{
			"agent_code": salesperson.AgentCode,
			"message":    "已存在代理码",
		})
	}

	// 生成新的代理码
	agentCode := utils.GenerateAgentCode()

	// 更新销售员信息
	if err := database.GetDB().Model(&salesperson).Update("agent_code", agentCode).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "更新代理码失败: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"agent_code": agentCode,
		"message":    "代理码生成成功",
	})
}

// CreateAgentInvitation 创建代理邀请
func CreateAgentInvitation(c *fiber.Ctx) error {
	// 获取当前销售员ID
	salespersonID, err := strconv.Atoi(c.Get("X-Salesperson-ID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 解析请求体
	var request struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败: " + err.Error(),
		})
	}

	// 验证必填字段
	if request.Email == "" && request.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "邮箱和电话至少填写一项",
		})
	}

	// 验证邮箱格式
	if request.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(request.Email) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "邮箱格式不正确",
			})
		}
	}

	// 验证电话号码格式
	if request.Phone != "" {
		phoneRegex := regexp.MustCompile(`^[0-9]{5,15}$`)
		if !phoneRegex.MatchString(request.Phone) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "电话号码格式不正确，应为5-15位数字",
			})
		}
	}

	// 查询销售员信息
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, salespersonID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败: " + err.Error(),
		})
	}

	// 检查是否已存在相同邮箱或电话的待处理邀请
	var existingInvitation models.SalespersonAgentInvitation
	query := database.GetDB().Where("status = ?", "pending")
	if request.Email != "" {
		query = query.Where("email = ?", request.Email)
	}
	if request.Phone != "" {
		query = query.Or("phone = ?", request.Phone)
	}

	if err := query.First(&existingInvitation).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "已存在相同邮箱或电话的待处理邀请",
		})
	} else if err != gorm.ErrRecordNotFound {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询邀请记录失败: " + err.Error(),
		})
	}

	// 生成邀请码
	inviteCode := utils.GenerateInviteCode()

	// 创建邀请记录
	invitation := models.SalespersonAgentInvitation{
		InviterID:  uint(salespersonID),
		InviteCode: inviteCode,
		Email:      request.Email,
		Phone:      request.Phone,
		Status:     "pending",
		ExpiredAt:  time.Now().AddDate(0, 0, 7), // 7天后过期
	}

	if err := database.GetDB().Create(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "创建邀请失败: " + err.Error(),
		})
	}

	// TODO: 发送邀请邮件或短信

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"invitation_id": invitation.ID,
		"invite_code":   inviteCode,
		"message":       "邀请创建成功",
	})
}

// AcceptAgentInvitation 接受代理邀请
func AcceptAgentInvitation(c *fiber.Ctx) error {
	// 解析请求体
	var request struct {
		InviteCode string `json:"invite_code"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "参数解析失败: " + err.Error(),
		})
	}

	// 验证必填字段
	if request.InviteCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "邀请码不能为空",
		})
	}

	// 查询邀请记录
	var invitation models.SalespersonAgentInvitation
	if err := database.GetDB().Where("invite_code = ? AND status = ?", request.InviteCode, "pending").First(&invitation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "邀请不存在或已失效",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询邀请失败: " + err.Error(),
		})
	}

	// 检查邀请是否过期
	if time.Now().After(invitation.ExpiredAt) {
		// 更新邀请状态为过期
		database.GetDB().Model(&invitation).Update("status", "expired")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "邀请已过期",
		})
	}

	// 获取当前销售员ID
	salespersonID, err := strconv.Atoi(c.Get("X-Salesperson-ID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 查询销售员信息
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, salespersonID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败: " + err.Error(),
		})
	}

	// 查询邀请人信息
	var inviter models.Salesperson
	if err := database.GetDB().First(&inviter, invitation.InviterID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "邀请人不存在",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询邀请人失败: " + err.Error(),
		})
	}

	// 检查代理层级是否超过限制
	if inviter.Level >= MaxAgentLevel {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "代理层级已达到最大限制，无法添加更多下级",
		})
	}

	// 检查是否已经有上级
	if salesperson.ParentID != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "您已经有上级代理，不能接受其他邀请",
		})
	}

	// 检查是否形成循环引用（防止A是B的上级，B又成为A的上级）
	if isCircularReference(invitation.InviterID, uint(salespersonID)) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "不能接受下级或间接下级的邀请，这会形成循环引用",
		})
	}

	// 检查新的层级是否会超过限制
	newLevel := inviter.Level + 1
	if newLevel > MaxAgentLevel {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("接受此邀请会使您的代理层级达到%d，超过最大限制%d", newLevel, MaxAgentLevel),
		})
	}

	// 开始事务
	tx := database.GetDB().Begin()

	// 使用defer确保事务在函数返回时被正确处理
	var txCommitted bool
	defer func() {
		// 如果事务还没有被提交，则回滚
		if !txCommitted && tx != nil {
			tx.Rollback()
			log.Println("事务已回滚")
		}
	}()

	// 更新销售员的上级关系
	salesperson.ParentID = &invitation.InviterID
	salesperson.Level = inviter.Level + 1
	if err := tx.Save(&salesperson).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "更新销售员关系失败: " + err.Error(),
		})
	}

	// 更新邀请人的下级数量
	if err := tx.Model(&models.Salesperson{}).Where("id = ?", inviter.ID).
		UpdateColumn("children_count", gorm.Expr("children_count + 1")).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "更新邀请人信息失败: " + err.Error(),
		})
	}

	// 更新邀请记录
	now := time.Now()
	invitation.Status = "accepted"
	invitation.InviteeID = &salesperson.ID
	invitation.AcceptedAt = &now
	if err := tx.Save(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "更新邀请记录失败: " + err.Error(),
		})
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "提交事务失败: " + err.Error(),
		})
	}

	// 标记事务已提交
	txCommitted = true

	return c.JSON(fiber.Map{
		"message": "成功接受邀请，已成为代理",
		"agent": fiber.Map{
			"id":   inviter.ID,
			"name": inviter.Name,
		},
	})
}

// isCircularReference 检查是否形成循环引用
// 检查potentialParentID是否是childID的下级或间接下级
func isCircularReference(potentialParentID, childID uint) bool {
	// 如果潜在的上级就是自己，直接返回true
	if potentialParentID == childID {
		return true
	}

	// 查询childID的所有直接下级
	var children []models.Salesperson
	if err := database.GetDB().Where("parent_id = ?", childID).Find(&children).Error; err != nil {
		log.Printf("查询下级失败: %v", err)
		return false // 查询失败时，为安全起见，不阻止操作
	}

	// 如果没有下级，则不会形成循环
	if len(children) == 0 {
		return false
	}

	// 检查直接下级
	for _, child := range children {
		if child.ID == potentialParentID {
			return true // 发现循环引用
		}

		// 递归检查间接下级
		if isCircularReference(potentialParentID, child.ID) {
			return true
		}
	}

	return false
}

// GetAgentHierarchy 获取代理层级结构
func GetAgentHierarchy(c *fiber.Ctx) error {
	// 获取当前销售员ID
	salespersonID, err := strconv.Atoi(c.Get("X-Salesperson-ID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 查询销售员信息
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, salespersonID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败: " + err.Error(),
		})
	}

	// 查询上级信息
	var parent *models.Salesperson
	if salesperson.ParentID != nil {
		var parentSalesperson models.Salesperson
		if err := database.GetDB().First(&parentSalesperson, *salesperson.ParentID).Error; err == nil {
			parent = &parentSalesperson
		}
	}

	// 查询下级信息
	var children []models.Salesperson
	if err := database.GetDB().Where("parent_id = ?", salesperson.ID).Find(&children).Error; err != nil {
		log.Printf("查询下级失败: %v", err)
		// 不返回错误，继续处理
	}

	// 构建响应
	response := fiber.Map{
		"id":             salesperson.ID,
		"name":           salesperson.Name,
		"level":          salesperson.Level,
		"children_count": salesperson.ChildrenCount,
		"agent_code":     salesperson.AgentCode,
	}

	if parent != nil {
		response["parent"] = fiber.Map{
			"id":   parent.ID,
			"name": parent.Name,
		}
	}

	if len(children) > 0 {
		childrenData := make([]fiber.Map, 0, len(children))
		for _, child := range children {
			childrenData = append(childrenData, fiber.Map{
				"id":             child.ID,
				"name":           child.Name,
				"level":          child.Level,
				"children_count": child.ChildrenCount,
			})
		}
		response["children"] = childrenData
	}

	return c.JSON(response)
}

// GetAgentCommissions 获取代理佣金记录
func GetAgentCommissions(c *fiber.Ctx) error {
	// 获取当前销售员ID
	salespersonID, err := strconv.Atoi(c.Get("X-Salesperson-ID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的销售员ID",
		})
	}

	// 查询销售员信息
	var salesperson models.Salesperson
	if err := database.GetDB().First(&salesperson, salespersonID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "销售员不存在",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询销售员失败: " + err.Error(),
		})
	}

	// 查询作为代理获得的佣金记录
	var agentCommissions []models.SalespersonAgentCommission
	if err := database.GetDB().Where("agent_id = ?", salespersonID).Find(&agentCommissions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询代理佣金记录失败: " + err.Error(),
		})
	}

	// 统计总佣金
	var totalCommission float64
	for _, commission := range agentCommissions {
		totalCommission += commission.CommissionAmount
	}

	return c.JSON(fiber.Map{
		"total_commission": totalCommission,
		"commissions":      agentCommissions,
	})
}

// ProcessAgentCommission 在销售记录创建后，处理代理佣金
// 支持多级代理分佣，每个上级都能获得相应的佣金
// 使用事务确保数据一致性
func ProcessAgentCommission(sale models.SalespersonSale, db *gorm.DB) error {
	// 查询销售员信息
	var salesperson models.Salesperson
	if err := db.First(&salesperson, sale.SalespersonID).Error; err != nil {
		return fmt.Errorf("查询销售员失败: %w", err)
	}

	// 如果没有上级，则不需要处理代理佣金
	if salesperson.ParentID == nil {
		return nil
	}

	// 开始事务
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开始事务失败: %w", tx.Error)
	}

	// 使用defer确保事务在函数返回时被正确处理
	var txCommitted bool
	defer func() {
		// 如果事务还没有被提交，则回滚
		if !txCommitted && tx != nil {
			tx.Rollback()
			log.Println("事务已回滚")
		}
	}()

	// 处理多级代理佣金
	// 从直接上级开始，逐级向上处理
	currentSalespersonID := sale.SalespersonID
	currentParentID := salesperson.ParentID
	currentLevel := salesperson.Level

	// 设置最大处理层级，防止无限循环
	maxLevels := MaxAgentLevel
	processedLevels := 0

	// 逐级向上处理佣金
	for currentParentID != nil && processedLevels < maxLevels {
		// 查询上级销售员
		var parent models.Salesperson
		if err := tx.First(&parent, *currentParentID).Error; err != nil {
			return fmt.Errorf("查询上级销售员(ID:%d)失败: %w", *currentParentID, err)
		}

		// 计算当前层级的佣金
		// 佣金比例随层级增加而递减
		var commissionRate float64
		var commissionAmount float64

		// 直接上级使用设置的佣金比例
		if currentLevel-1 == parent.Level {
			commissionRate = parent.ParentCommissionRate
		} else {
			// 间接上级佣金比例递减
			// 每上升一级，佣金比例减半
			levelDiff := currentLevel - parent.Level - 1
			divisor := math.Pow(2, float64(levelDiff))
			commissionRate = parent.ParentCommissionRate / divisor
		}

		// 计算佣金金额
		commissionAmount = sale.SaleAmount * commissionRate

		// 如果佣金金额太小，则不再处理
		if commissionAmount < 0.01 {
			break
		}

		// 创建代理佣金记录
		agentCommission := models.SalespersonAgentCommission{
			SaleID:           sale.ID,
			SalespersonID:    currentSalespersonID,
			AgentID:          parent.ID,
			AgentLevel:       parent.Level,
			OriginalAmount:   sale.SaleAmount,
			CommissionRate:   commissionRate,
			CommissionAmount: commissionAmount,
			Status:           "pending",
		}

		if err := tx.Create(&agentCommission).Error; err != nil {
			return fmt.Errorf("创建代理佣金记录失败: %w", err)
		}

		// 更新上级销售员的总佣金
		if err := tx.Model(&models.Salesperson{}).Where("id = ?", parent.ID).
			UpdateColumn("total_commission", gorm.Expr("total_commission + ?", commissionAmount)).Error; err != nil {
			return fmt.Errorf("更新上级销售员佣金失败: %w", err)
		}

		// 准备处理下一级
		currentSalespersonID = parent.ID
		currentParentID = parent.ParentID
		currentLevel = parent.Level
		processedLevels++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 标记事务已提交
	txCommitted = true

	return nil
}
