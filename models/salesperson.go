package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Salesperson 销售员模型
// 用于存储销售员的基本信息，包括姓名、联系方式、账号等
type Salesperson struct {
	ID                   uint       `json:"id" gorm:"primaryKey"`                      // 主键ID
	Username             string     `json:"username" gorm:"size:50;uniqueIndex"`       // 用户名，登录用，唯一
	Password             string     `json:"-" gorm:"size:100"`                         // 密码，不返回给前端
	Name                 string     `json:"name" gorm:"size:50"`                       // 姓名
	Phone                string     `json:"phone" gorm:"size:20"`                      // 电话
	Email                string     `json:"email" gorm:"size:100"`                     // 邮箱
	Status               string     `json:"status" gorm:"size:20;default:active"`      // 状态：active在职, inactive离职, suspended暂停
	Avatar               string     `json:"avatar" gorm:"size:255"`                    // 头像URL
	CommissionRate       float64    `json:"commission_rate" gorm:"default:0"`          // 默认佣金比例，例如0.1表示10%
	TotalSales           float64    `json:"total_sales" gorm:"default:0"`              // 总销售额
	TotalCommission      float64    `json:"total_commission" gorm:"default:0"`         // 总佣金
	CreatorID            uint       `json:"creator_id" gorm:"not null"`                // 创建者ID，记录谁创建了这个销售员
	ParentID             *uint      `json:"parent_id" gorm:"index"`                    // 上级销售员ID，允许为空
	Level                int        `json:"level" gorm:"default:0"`                    // 代理层级，0表示顶级代理
	ChildrenCount        int        `json:"children_count" gorm:"default:0"`           // 下级销售员数量
	AgentCode            string     `json:"agent_code" gorm:"size:50;uniqueIndex"`     // 代理邀请码，用于发展下线
	ParentCommissionRate float64    `json:"parent_commission_rate" gorm:"default:0.1"` // 上级提成比例，默认10%
	LastLoginAt          *time.Time `json:"last_login_at"`                             // 最后登录时间
	CreatedAt            time.Time  `json:"created_at" gorm:"autoCreateTime"`          // 创建时间
	UpdatedAt            time.Time  `json:"updated_at" gorm:"autoUpdateTime"`          // 更新时间
}

// TableName 返回表名
func (Salesperson) TableName() string {
	return "salespersons"
}

// SetPassword 设置加密密码
func (s *Salesperson) SetPassword(plainPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.Password = string(hashedPassword)
	return nil
}

// CheckPassword 验证密码
func (s *Salesperson) CheckPassword(plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(s.Password), []byte(plainPassword))
	return err == nil
}

// SalespersonQuery 销售员查询参数
type SalespersonQuery struct {
	Username  string `json:"username" query:"username"`     // 用户名
	Name      string `json:"name" query:"name"`             // 姓名
	Status    string `json:"status" query:"status"`         // 状态
	CreatorID uint   `json:"creator_id" query:"creator_id"` // 创建者ID
	Page      int    `json:"page" query:"page"`             // 页码
	PageSize  int    `json:"page_size" query:"page_size"`   // 每页数量
}

// SalespersonProduct 销售员可销售产品关联
// 记录销售员可以销售哪些软件的哪些卡密类型
type SalespersonProduct struct {
	ID             uint      `json:"id" gorm:"primaryKey"`                                // 主键ID
	SalespersonID  uint      `json:"salesperson_id" gorm:"index:idx_salesperson_product"` // 销售员ID
	SoftwareID     uint      `json:"software_id" gorm:"index:idx_salesperson_product"`    // 软件ID
	KeyTypeID      uint      `json:"key_type_id" gorm:"index:idx_salesperson_product"`    // 卡密类型ID
	CommissionRate float64   `json:"commission_rate"`                                     // 特定产品的佣金比例，覆盖销售员默认佣金比例
	KeyGenLimit    int       `json:"key_gen_limit" gorm:"default:0"`                      // 卡密生成数量限制，0表示无限制
	KeysGenerated  int       `json:"keys_generated" gorm:"default:0"`                     // 已生成卡密数量
	IsActive       bool      `json:"is_active" gorm:"default:true"`                       // 是否启用
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`                    // 创建时间
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`                    // 更新时间
}

// TableName 返回表名
func (SalespersonProduct) TableName() string {
	return "salesperson_products"
}

// SalespersonSale 销售员销售记录
// 记录销售员的每一笔销售记录
type SalespersonSale struct {
	ID             uint       `json:"id" gorm:"primaryKey"`                             // 主键ID
	SalespersonID  uint       `json:"salesperson_id" gorm:"index:idx_salesperson_sale"` // 销售员ID
	KeyID          uint       `json:"key_id" gorm:"index:idx_salesperson_sale"`         // 卡密ID
	SoftwareID     uint       `json:"software_id"`                                      // 软件ID
	KeyTypeID      uint       `json:"key_type_id"`                                      // 卡密类型ID
	CustomerName   string     `json:"customer_name" gorm:"size:100"`                    // 客户姓名
	CustomerPhone  string     `json:"customer_phone" gorm:"size:20"`                    // 客户电话
	CustomerEmail  string     `json:"customer_email" gorm:"size:100"`                   // 客户邮箱
	SaleAmount     float64    `json:"sale_amount"`                                      // 销售金额
	CommissionRate float64    `json:"commission_rate"`                                  // 实际佣金比例
	Commission     float64    `json:"commission"`                                       // 实际佣金金额
	Status         string     `json:"status" gorm:"default:pending"`                    // 状态：pending待结算, settled已结算, cancelled已取消
	SettledAt      *time.Time `json:"settled_at"`                                       // 结算时间
	Notes          string     `json:"notes" gorm:"type:text"`                           // 备注
	CreatedAt      time.Time  `json:"created_at" gorm:"autoCreateTime"`                 // 创建时间
	UpdatedAt      time.Time  `json:"updated_at" gorm:"autoUpdateTime"`                 // 更新时间
}

// TableName 返回表名
func (SalespersonSale) TableName() string {
	return "salesperson_sales"
}

// SalespersonCustomer 销售员客户关系
// 记录销售员与客户的绑定关系
type SalespersonCustomer struct {
	ID            uint      `json:"id" gorm:"primaryKey"`                                 // 主键ID
	SalespersonID uint      `json:"salesperson_id" gorm:"index:idx_salesperson_customer"` // 销售员ID
	CustomerID    uint      `json:"customer_id" gorm:"index:idx_salesperson_customer"`    // 客户ID
	CustomerName  string    `json:"customer_name"`                                        // 客户姓名
	CustomerPhone string    `json:"customer_phone"`                                       // 客户电话
	CustomerEmail string    `json:"customer_email"`                                       // 客户邮箱
	Status        string    `json:"status" gorm:"default:active"`                         // 状态：active活跃, inactive非活跃
	Notes         string    `json:"notes" gorm:"type:text"`                               // 备注
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`                     // 创建时间
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`                     // 更新时间
}

// TableName 返回表名
func (SalespersonCustomer) TableName() string {
	return "salesperson_customers"
}

// SalespersonCommissionSettlement 销售员佣金结算记录
// 记录销售员的佣金结算情况
type SalespersonCommissionSettlement struct {
	ID              uint       `json:"id" gorm:"primaryKey"`                                          // 主键ID
	SalespersonID   uint       `json:"salesperson_id" gorm:"index"`                                   // 销售员ID
	SettlementNo    string     `json:"settlement_no" gorm:"uniqueIndex:idx_settlement_no,length:191"` // 结算单号
	StartDate       time.Time  `json:"start_date"`                                                    // 结算周期开始日期
	EndDate         time.Time  `json:"end_date"`                                                      // 结算周期结束日期
	TotalSales      float64    `json:"total_sales"`                                                   // 总销售额
	TotalCommission float64    `json:"total_commission"`                                              // 总佣金
	Status          string     `json:"status" gorm:"default:pending"`                                 // 状态：pending待支付, paid已支付, cancelled已取消
	PaymentMethod   string     `json:"payment_method"`                                                // 支付方式
	PaymentRef      string     `json:"payment_ref"`                                                   // 支付参考号
	PaidAmount      float64    `json:"paid_amount"`                                                   // 实际支付金额
	PaidAt          *time.Time `json:"paid_at"`                                                       // 支付时间
	ApproverID      uint       `json:"approver_id"`                                                   // 审批人ID
	Notes           string     `json:"notes" gorm:"type:text"`                                        // 备注
	CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`                              // 创建时间
	UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`                              // 更新时间
}

// TableName 返回表名
func (SalespersonCommissionSettlement) TableName() string {
	return "salesperson_commission_settlements"
}
