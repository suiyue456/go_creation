// Package models 定义了应用程序的数据模型
package models

import (
	"time"
)

// SoftwareKeyType 软件与卡密类型的关联模型
// 用于建立软件和卡密类型之间的多对多关系
type SoftwareKeyType struct {
	ID         uint      `json:"id" gorm:"primaryKey"`              // 主键ID
	SoftwareID uint      `json:"software_id" gorm:"not null;index"` // 软件ID，关联到Software表
	KeyTypeID  uint      `json:"key_type_id" gorm:"not null;index"` // 卡密类型ID，关联到KeyType表
	IsActive   bool      `json:"is_active" gorm:"default:true"`     // 是否启用，控制该关联是否有效
	CreatorID  uint      `json:"creator_id" gorm:"not null"`        // 创建者ID，记录谁创建了这个关联
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`  // 创建时间，记录关联的创建时间
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`  // 更新时间，记录关联的最后更新时间
}

// TableName 返回表名
// GORM会使用此方法来确定模型对应的数据库表名
func (SoftwareKeyType) TableName() string {
	return "software_key_types"
}

// SoftwareKeyTypeQuery 软件与卡密类型关联的查询参数
// 用于接收前端传来的查询条件，进行关联关系的筛选查询
type SoftwareKeyTypeQuery struct {
	SoftwareID uint  `json:"software_id" query:"software_id"` // 软件ID，用于按软件筛选
	KeyTypeID  uint  `json:"key_type_id" query:"key_type_id"` // 卡密类型ID，用于按卡密类型筛选
	IsActive   *bool `json:"is_active" query:"is_active"`     // 是否启用，用于按启用状态筛选
	CreatorID  uint  `json:"creator_id" query:"creator_id"`   // 创建者ID，用于按创建者筛选
	Page       int   `json:"page" query:"page"`               // 页码，用于分页查询
	Limit      int   `json:"limit" query:"limit"`             // 每页数量，用于分页查询
}

// CreateSoftwareKeyTypeRequest 创建软件与卡密类型关联的请求参数
// 用于接收前端传来的创建关联关系的数据
type CreateSoftwareKeyTypeRequest struct {
	SoftwareID uint `json:"software_id" validate:"required"` // 软件ID，必填
	KeyTypeID  uint `json:"key_type_id" validate:"required"` // 卡密类型ID，必填
}

// BeforeCreate GORM的钩子函数，在创建记录前执行
// 用于设置默认值和执行数据验证
func (skt *SoftwareKeyType) BeforeCreate() error {
	// 设置默认状态
	if !skt.IsActive {
		skt.IsActive = true // 默认为启用状态
	}

	// 验证数据有效性
	return skt.Validate()
}

// Enable 启用关联
// 设置关联状态为启用
func (skt *SoftwareKeyType) Enable() {
	skt.IsActive = true
}

// Disable 禁用关联
// 设置关联状态为禁用
func (skt *SoftwareKeyType) Disable() {
	skt.IsActive = false
}

// IsEnabled 检查关联是否启用
// 返回：
//   - bool: true表示关联已启用，false表示关联已禁用
func (skt *SoftwareKeyType) IsEnabled() bool {
	return skt.IsActive
}

// Validate 验证关联数据的有效性
// 返回：
//   - error: 如果验证失败，返回错误信息；验证通过返回nil
func (skt *SoftwareKeyType) Validate() error {
	// TODO: 实现关联数据的验证逻辑
	return nil
}
