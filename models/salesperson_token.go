package models

import (
	"time"
)

// SalespersonToken 销售员登录令牌模型
// 该模型用于存储销售员的JWT认证令牌及相关会话信息
// 支持多设备登录，每个设备会创建独立的令牌记录
// 包含令牌本身、设备信息、IP地址和过期时间等安全相关字段
type SalespersonToken struct {
	ID            uint      `json:"id" gorm:"primaryKey"`             // 主键ID
	SalespersonID uint      `json:"salesperson_id" gorm:"index"`      // 关联的销售员ID，添加索引以提高查询性能
	Token         string    `json:"token" gorm:"size:500;index"`      // JWT令牌字符串，添加索引以提高查询性能
	UserAgent     string    `json:"user_agent" gorm:"size:255"`       // 用户代理信息，用于识别登录设备
	IP            string    `json:"ip" gorm:"size:50"`                // 登录IP地址，用于安全审计
	ExpiredAt     time.Time `json:"expired_at" gorm:"index"`          // 令牌过期时间，添加索引以提高查询性能
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"` // 记录创建时间，自动设置
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"` // 记录更新时间，自动更新
}

// TableName 返回表名
// 自定义表名为salesperson_tokens，符合数据库命名规范
// 该方法实现了gorm.Tabler接口，用于指定模型对应的数据库表名
func (SalespersonToken) TableName() string {
	return "salesperson_tokens"
}
