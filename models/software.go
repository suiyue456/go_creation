package models

import (
	"time"
)

// Package models 定义了应用程序的数据模型

// Software 软件模型
// 用于存储软件的基本信息，包括名称、描述、版本、公告等
type Software struct {
	ID           uint      `json:"id" gorm:"primaryKey"`                          // 主键ID
	Name         string    `json:"name" gorm:"size:100;not null"`                 // 软件名称，不能为空
	Description  string    `json:"description" gorm:"type:text"`                  // 软件描述，详细说明软件的功能和特点
	Version      string    `json:"version" gorm:"size:50"`                        // 软件版本号，如"1.0.0"
	Announcement string    `json:"announcement" gorm:"type:text"`                 // 软件公告，用于向用户展示重要信息
	Status       string    `json:"status" gorm:"size:20;default:active"`          // 软件状态：active活跃, inactive非活跃
	IsActive     bool      `json:"is_active" gorm:"default:true"`                 // 是否启用，控制软件是否可用
	CreatorID    uint      `json:"creator_id" gorm:"not null"`                    // 创建者ID，记录谁创建了这个软件
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`              // 创建时间，记录软件的创建时间
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`              // 更新时间，记录软件的最后更新时间
	KeyTypes     []KeyType `json:"key_types" gorm:"many2many:software_key_types"` // 关联的密钥类型，多对多关系
}

// TableName 返回表名
// GORM会使用此方法来确定模型对应的数据库表名
func (Software) TableName() string {
	return "softwares"
}

// SoftwareQuery 软件查询参数
// 用于接收前端传来的查询条件，进行软件的筛选查询
type SoftwareQuery struct {
	Name      string `json:"name" query:"name"`             // 软件名称，用于按名称筛选软件
	Status    string `json:"status" query:"status"`         // 状态，用于按状态筛选软件
	IsActive  *bool  `json:"is_active" query:"is_active"`   // 是否启用，用于按启用状态筛选
	CreatorID uint   `json:"creator_id" query:"creator_id"` // 创建者ID，用于按创建者筛选软件
	Page      int    `json:"page" query:"page"`             // 页码，用于分页查询
	Limit     int    `json:"limit" query:"limit"`           // 每页数量，用于分页查询
}

// Validate 验证软件数据的有效性
// 返回：
//   - error: 如果验证失败，返回错误信息；验证通过返回nil
func (s *Software) Validate() error {
	// TODO: 实现软件数据的验证逻辑
	return nil
}

// BeforeCreate GORM的钩子函数，在创建记录前执行
// 用于设置默认值和执行数据验证
func (s *Software) BeforeCreate() error {
	// 设置默认状态
	if s.Status == "" {
		s.Status = "active" // 默认为活跃状态
	}
	if !s.IsActive {
		s.IsActive = true // 默认为启用状态
	}

	// 验证数据有效性
	return s.Validate()
}

// Enable 启用软件
// 设置软件状态为启用
func (s *Software) Enable() {
	s.Status = "active"
	s.IsActive = true
}

// Disable 禁用软件
// 设置软件状态为禁用
func (s *Software) Disable() {
	s.Status = "inactive"
	s.IsActive = false
}

// IsEnabled 检查软件是否启用
// 返回：
//   - bool: true表示软件已启用，false表示软件已禁用
func (s *Software) IsEnabled() bool {
	return s.Status == "active" && s.IsActive
}

// AddKeyType 添加密钥类型
// 参数：
//   - keyType: 要添加的密钥类型
func (s *Software) AddKeyType(keyType KeyType) {
	s.KeyTypes = append(s.KeyTypes, keyType)
}

// RemoveKeyType 移除密钥类型
// 参数：
//   - keyTypeID: 要移除的密钥类型ID
func (s *Software) RemoveKeyType(keyTypeID uint) {
	for i, kt := range s.KeyTypes {
		if kt.ID == keyTypeID {
			s.KeyTypes = append(s.KeyTypes[:i], s.KeyTypes[i+1:]...)
			break
		}
	}
}

// HasKeyType 检查是否包含指定的密钥类型
// 参数：
//   - keyTypeID: 要检查的密钥类型ID
//
// 返回：
//   - bool: true表示包含该密钥类型，false表示不包含
func (s *Software) HasKeyType(keyTypeID uint) bool {
	for _, kt := range s.KeyTypes {
		if kt.ID == keyTypeID {
			return true
		}
	}
	return false
}

// CreateSoftwareRequest 创建软件的请求参数
// 用于接收前端传来的创建软件的数据
type CreateSoftwareRequest struct {
	Name         string `json:"name" validate:"required"`    // 软件名称，必填
	Description  string `json:"description"`                 // 软件描述
	Version      string `json:"version" validate:"required"` // 软件版本号，必填
	Announcement string `json:"announcement"`                // 软件公告
}
