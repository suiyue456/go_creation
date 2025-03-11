// Package models 定义了应用程序的数据模型
// 包含所有与数据库表对应的结构体定义和相关方法
package models

import (
	"time"
)

// Key 表示软件授权密钥
// 该结构体对应数据库中的keys表
type Key struct {
	ID            uint       `json:"id" gorm:"primaryKey"`                          // 主键ID
	Code          string     `json:"code" gorm:"uniqueIndex;size:64"`               // 密钥代码，唯一索引
	KeyCode       string     `json:"key_code" gorm:"uniqueIndex;size:32"`           // 激活码，唯一索引
	TypeID        uint       `json:"type_id"`                                       // 卡密类型ID
	TypeName      string     `json:"type_name" gorm:"size:100"`                     // 卡密类型名称
	Hours         int        `json:"hours"`                                         // 有效期小时数
	Price         float64    `json:"price"`                                         // 价格
	SoftwareID    uint       `json:"software_id"`                                   // 软件ID
	SoftwareName  string     `json:"software_name" gorm:"size:100"`                 // 软件名称
	Status        string     `json:"status" gorm:"type:varchar(20);default:unused"` // 状态：unused,used,void
	CreatorID     uint       `json:"creator_id"`                                    // 创建者ID
	CreatorType   string     `json:"creator_type" gorm:"size:20"`                   // 创建者类型
	SalespersonID uint       `json:"salesperson_id"`                                // 销售员ID
	UserID        *uint      `json:"user_id"`                                       // 使用者ID
	DeviceInfo    string     `json:"device_info" gorm:"type:text"`                  // 设备信息
	UsedAt        *time.Time `json:"used_at"`                                       // 使用时间
	ExpiredAt     *time.Time `json:"expired_at"`                                    // 过期时间
	ActivatedAt   *time.Time `json:"activated_at"`                                  // 激活时间
	IsBlacklisted bool       `json:"is_blacklisted" gorm:"default:false"`           // 是否黑名单
	CreatedAt     time.Time  `json:"created_at"`                                    // 创建时间
	UpdatedAt     time.Time  `json:"updated_at"`                                    // 更新时间
}

// TableName 指定模型对应的数据库表名
func (Key) TableName() string {
	return "keys"
}

// Validate 验证密钥数据的有效性
// 返回：
//   - error: 如果验证失败，返回错误信息；验证通过返回nil
func (k *Key) Validate() error {
	// TODO: 实现密钥数据的验证逻辑
	return nil
}

// BeforeCreate GORM的钩子函数，在创建记录前执行
// 用于设置默认值和执行数据验证
func (k *Key) BeforeCreate() error {
	// 设置默认状态
	if k.Status == "" {
		k.Status = "unused"
	}

	// 验证数据有效性
	return k.Validate()
}

// Activate 激活密钥
// 设置密钥状态为已使用，并记录激活时间
func (k *Key) Activate() error {
	now := time.Now()
	k.Status = "used"
	k.ActivatedAt = &now
	k.UsedAt = &now
	return nil
}

// Disable 禁用密钥
// 设置密钥状态为已禁用
// 返回：
//   - error: 如果禁用失败，返回错误信息；禁用成功返回nil
func (k *Key) Disable() error {
	k.Status = "void"
	// TODO: 添加禁用逻辑，如记录日志等
	return nil
}

// IsValid 检查密钥是否有效
// 检查密钥的状态和有效期
// 返回：
//   - bool: true表示密钥有效，false表示密钥无效
func (k *Key) IsValid() bool {
	// 检查状态
	if k.Status != "used" {
		return false
	}

	// 检查是否过期
	if k.ExpiredAt != nil && time.Now().After(*k.ExpiredAt) {
		return false
	}

	return true
}

// KeyQuery 卡密查询参数
type KeyQuery struct {
	Page          int    `query:"page"`           // 页码
	PageSize      int    `query:"page_size"`      // 每页数量
	Status        string `query:"status"`         // 状态筛选
	TypeID        uint   `query:"type_id"`        // 类型ID筛选
	SoftwareID    uint   `query:"software_id"`    // 软件ID筛选
	Code          string `query:"code"`           // 卡密码筛选
	KeyCode       string `query:"key_code"`       // 激活码筛选
	CreatorID     uint   `query:"creator_id"`     // 创建者ID筛选
	CreatorType   string `query:"creator_type"`   // 创建者类型筛选
	SalespersonID uint   `query:"salesperson_id"` // 销售员ID筛选
	UserID        uint   `query:"user_id"`        // 使用者ID筛选
	ActivatorID   uint   `query:"activator_id"`   // 激活者ID筛选
	StartTime     string `query:"start_time"`     // 开始时间
	EndTime       string `query:"end_time"`       // 结束时间
	SortBy        string `query:"sort_by"`        // 排序字段
	SortOrder     string `query:"sort_order"`     // 排序方式
}
