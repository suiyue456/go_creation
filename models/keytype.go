// Package models 定义了应用程序的数据模型
package models

import (
	"time"
)

// KeyType 卡密类型模型
// 用于定义不同类型的卡密，包括名称、描述、有效期、价格等属性
type KeyType struct {
	ID          uint       `gorm:"primaryKey" json:"id"`                                  // 主键ID
	Name        string     `gorm:"column:name;not null" json:"name"`                      // 类型名称，如"月卡"、"年卡"等
	Description string     `gorm:"column:description;type:text" json:"description"`       // 类型描述，详细说明卡密类型的用途和特点
	Hours       int        `gorm:"column:hours" json:"hours"`                             // 有效期（小时），表示该类型卡密的有效时长
	Price       float64    `gorm:"column:price" json:"price"`                             // 价格，表示该类型卡密的售价
	Status      string     `gorm:"column:status;default:active" json:"status"`            // 状态：active活跃, inactive非活跃
	IsActive    bool       `gorm:"column:is_active;default:true" json:"is_active"`        // 是否启用，控制该类型卡密是否可用
	IsUniversal bool       `gorm:"column:is_universal;default:false" json:"is_universal"` // 是否为通用卡密，通用卡密可用于多个软件
	CreatorID   uint       `gorm:"column:creator_id" json:"creator_id"`                   // 创建者ID（默认为admin），记录谁创建了这个卡密类型
	SellerID    uint       `gorm:"column:seller_id" json:"seller_id"`                     // 销售员ID，记录哪个销售员负责销售这类卡密
	Software    []Software `gorm:"many2many:software_key_types" json:"software"`          // 关联的软件，多对多关系
	CreatedAt   time.Time  `json:"created_at"`                                            // 创建时间，记录卡密类型的创建时间
	UpdatedAt   time.Time  `json:"updated_at"`                                            // 更新时间，记录卡密类型的最后更新时间
}

// TableName 返回表名
// GORM会使用此方法来确定模型对应的数据库表名
func (KeyType) TableName() string {
	return "key_types"
}

// BeforeCreate GORM的钩子函数，在创建记录前执行
// 用于设置默认值和执行数据验证
func (kt *KeyType) BeforeCreate() error {
	// 设置默认状态
	if kt.Status == "" {
		kt.Status = "active" // 默认为活跃状态
	}
	if !kt.IsActive {
		kt.IsActive = true // 默认为启用状态
	}

	// 验证数据有效性
	return kt.Validate()
}

// Enable 启用密钥类型
// 设置密钥类型状态为启用
func (kt *KeyType) Enable() {
	kt.Status = "active"
	kt.IsActive = true
}

// Disable 禁用密钥类型
// 设置密钥类型状态为禁用
func (kt *KeyType) Disable() {
	kt.Status = "inactive"
	kt.IsActive = false
}

// IsEnabled 检查密钥类型是否启用
// 返回：
//   - bool: true表示密钥类型已启用，false表示密钥类型已禁用
func (kt *KeyType) IsEnabled() bool {
	return kt.Status == "active" && kt.IsActive
}

// AddSoftware 添加软件关联
// 参数：
//   - software: 要添加的软件
func (kt *KeyType) AddSoftware(software Software) {
	kt.Software = append(kt.Software, software)
}

// RemoveSoftware 移除软件关联
// 参数：
//   - softwareID: 要移除的软件ID
func (kt *KeyType) RemoveSoftware(softwareID uint) {
	for i, s := range kt.Software {
		if s.ID == softwareID {
			kt.Software = append(kt.Software[:i], kt.Software[i+1:]...)
			break
		}
	}
}

// HasSoftware 检查是否包含指定的软件
// 参数：
//   - softwareID: 要检查的软件ID
//
// 返回：
//   - bool: true表示包含该软件，false表示不包含
func (kt *KeyType) HasSoftware(softwareID uint) bool {
	for _, s := range kt.Software {
		if s.ID == softwareID {
			return true
		}
	}
	return false
}

// Validate 验证密钥类型数据的有效性
// 返回：
//   - error: 如果验证失败，返回错误信息；验证通过返回nil
func (kt *KeyType) Validate() error {
	// TODO: 实现密钥类型数据的验证逻辑
	return nil
}

// KeyTypeQuery 卡密类型查询参数
// 用于接收前端传来的查询条件，进行卡密类型的筛选查询
type KeyTypeQuery struct {
	Name        string `json:"name" query:"name"`                 // 类型名称，用于按名称筛选
	Status      string `json:"status" query:"status"`             // 状态，用于按状态筛选
	IsActive    *bool  `json:"is_active" query:"is_active"`       // 是否启用，用于按启用状态筛选
	IsUniversal *bool  `json:"is_universal" query:"is_universal"` // 是否为通用卡密，用于按通用性筛选
	CreatorID   uint   `json:"creator_id" query:"creator_id"`     // 创建者ID，用于按创建者筛选
	SellerID    uint   `json:"seller_id" query:"seller_id"`       // 销售员ID，用于按销售员筛选
	Page        int    `json:"page" query:"page"`                 // 页码，用于分页查询
	Limit       int    `json:"limit" query:"limit"`               // 每页数量，用于分页查询
}

// CreateKeyTypeRequest 创建卡密类型的请求参数
// 用于接收前端传来的创建卡密类型的数据
type CreateKeyTypeRequest struct {
	Name        string  `json:"name" validate:"required"`        // 类型名称，必填
	Description string  `json:"description"`                     // 类型描述
	Hours       int     `json:"hours" validate:"required,min=1"` // 有效期（小时），必填且大于0
	Price       float64 `json:"price" validate:"required,min=0"` // 价格，必填且不小于0
	IsUniversal bool    `json:"is_universal"`                    // 是否为通用卡密
	SellerID    uint    `json:"seller_id"`                       // 销售员ID
}
