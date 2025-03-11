package models

import (
	"time"
)

// SalespersonAgentCommission 销售员代理佣金记录
// 记录上下级销售员之间的佣金分成记录
type SalespersonAgentCommission struct {
	ID               uint      `json:"id" gorm:"primaryKey"`             // 主键ID
	SaleID           uint      `json:"sale_id" gorm:"index"`             // 销售记录ID
	SalespersonID    uint      `json:"salesperson_id" gorm:"index"`      // 销售员ID（下级）
	AgentID          uint      `json:"agent_id" gorm:"index"`            // 代理ID（上级）
	AgentLevel       int       `json:"agent_level"`                      // 代理层级
	OriginalAmount   float64   `json:"original_amount"`                  // 原始销售金额
	CommissionRate   float64   `json:"commission_rate"`                  // 佣金比例
	CommissionAmount float64   `json:"commission_amount"`                // 佣金金额
	Status           string    `json:"status" gorm:"default:pending"`    // 状态：pending待结算, settled已结算, cancelled已取消
	SettlementID     *uint     `json:"settlement_id"`                    // 结算单ID
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"` // 创建时间
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"` // 更新时间
}

// TableName 返回表名
func (SalespersonAgentCommission) TableName() string {
	return "salesperson_agent_commissions"
}

// SalespersonAgentInvitation 销售员代理邀请记录
// 记录销售员邀请其他销售员加入的记录
type SalespersonAgentInvitation struct {
	ID         uint       `json:"id" gorm:"primaryKey"`                   // 主键ID
	InviterID  uint       `json:"inviter_id" gorm:"index"`                // 邀请人ID
	InviteeID  *uint      `json:"invitee_id"`                             // 被邀请人ID，注册后才有
	InviteCode string     `json:"invite_code" gorm:"size:50;uniqueIndex"` // 邀请码
	Email      string     `json:"email" gorm:"size:100"`                  // 被邀请人邮箱
	Phone      string     `json:"phone" gorm:"size:20"`                   // 被邀请人电话
	Status     string     `json:"status" gorm:"default:pending"`          // 状态：pending待接受, accepted已接受, rejected已拒绝, expired已过期
	AcceptedAt *time.Time `json:"accepted_at"`                            // 接受时间
	ExpiredAt  time.Time  `json:"expired_at"`                             // 过期时间
	CreatedAt  time.Time  `json:"created_at" gorm:"autoCreateTime"`       // 创建时间
	UpdatedAt  time.Time  `json:"updated_at" gorm:"autoUpdateTime"`       // 更新时间
}

// TableName 返回表名
func (SalespersonAgentInvitation) TableName() string {
	return "salesperson_agent_invitations"
}
