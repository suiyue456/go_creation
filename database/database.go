// Package database 提供数据库连接和管理功能
// 该包负责处理与数据库相关的所有操作，包括：
// - 数据库连接的建立和管理
// - 连接池的配置
// - 数据库迁移
// - 提供全局数据库实例
package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"go_creation/models"
)

// DB 全局数据库连接实例
// 这个变量在整个应用程序中被共享使用
// 通过 GetDB() 函数安全地访问
var DB *gorm.DB

// GetDB 返回数据库连接实例
// 这个函数是获取数据库连接的推荐方式
// 它确保了数据库连接的线程安全访问
func GetDB() *gorm.DB {
	return DB
}

// SetDB 设置数据库连接
// 主要用于测试场景，允许注入模拟的数据库连接
// 参数:
//   - newDB: 新的数据库连接实例
func SetDB(newDB *gorm.DB) {
	DB = newDB
}

// Init 初始化数据库模块
// 该函数执行以下操作：
// 1. 加载环境变量
// 2. 建立数据库连接
// 3. 配置连接池
// 4. 设置字符集和排序规则
func Init() {
	// 加载.env文件中的环境变量
	// 如果文件不存在或无法加载，程序会终止
	if err := godotenv.Load(); err != nil {
		log.Fatal("加载.env文件失败")
	}

	// 初始化数据库连接
	initConnection()
}

// initConnection 初始化数据库连接
// 该函数负责：
// 1. 从环境变量获取数据库配置
// 2. 配置GORM日志
// 3. 建立数据库连接
// 4. 配置连接池参数
// 5. 设置数据库默认字符集
func initConnection() {
	// 从环境变量获取数据库配置
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// 配置GORM日志
	// 设置日志级别、慢查询阈值等
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second, // 慢查询阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略记录未找到的错误
			Colorful:                  true,        // 启用彩色输出
		},
	)

	// 先尝试连接MySQL服务器（不指定数据库）
	// 这样可以在数据库不存在时创建它
	dsnWithoutDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port)

	tempDB, err := gorm.Open(mysql.Open(dsnWithoutDB), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接MySQL服务器失败: %v", err)
	}

	// 创建数据库（如果不存在）
	// 使用utf8mb4字符集和unicode_ci排序规则
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbname)
	if err := tempDB.Exec(createDBSQL).Error; err != nil {
		log.Fatalf("创建数据库失败: %v", err)
	}

	// 构建完整的数据库连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&collation=utf8mb4_unicode_ci",
		user, password, host, port, dbname)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	// 获取底层的sqlDB以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("无法获取底层数据库连接: %v", err)
	}

	// 设置连接池参数
	// 这些参数需要根据实际负载情况调整
	sqlDB.SetMaxOpenConns(25)                  // 最大打开连接数
	sqlDB.SetMaxIdleConns(10)                  // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最大生存时间
	sqlDB.SetConnMaxIdleTime(30 * time.Minute) // 空闲连接最大生存时间

	// 设置数据库默认字符集和排序规则
	db.Exec("SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci")
	db.Exec("SET CHARACTER SET utf8mb4")
	db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")

	DB = db
	log.Printf("数据库已成功连接到 %s:%s/%s", host, port, dbname)
}

// Migrate 执行数据库迁移
// 该函数使用GORM的AutoMigrate功能自动创建或更新数据库表
// 它会：
// 1. 创建不存在的表
// 2. 添加缺少的字段
// 3. 更新字段类型
// 4. 添加缺少的索引
func Migrate() {
	log.Println("开始数据库迁移...")

	// 配置GORM自动迁移选项
	db := DB.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")

	// 执行自动迁移
	// 需要迁移的模型按照依赖关系排序
	err := db.AutoMigrate(
		// 基础模型
		&models.KeyType{},
		&models.Key{},
		&models.Software{},
		&models.SoftwareKeyType{},
		// 销售员相关模型
		&models.Salesperson{},
		&models.SalespersonProduct{},
		&models.SalespersonSale{},
		&models.SalespersonCustomer{},
		&models.SalespersonCommissionSettlement{},
		&models.SalespersonToken{},
		// 代理相关模型
		&models.SalespersonAgentCommission{},
		&models.SalespersonAgentInvitation{},
	)

	if err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	log.Println("数据库迁移成功")
}
