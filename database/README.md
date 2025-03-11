# 数据库模块

本模块负责处理应用程序的数据库连接、初始化和迁移。

## 文件结构

- `database.go`: 包含所有数据库相关的功能，包括初始化连接、数据库迁移和提供数据库连接实例。

## 使用方法

### 初始化数据库

在应用程序启动时，需要调用 `database.Init()` 函数来初始化数据库连接：

```go
import "go_creation/database"

func main() {
    // 初始化数据库连接
    database.Init()
    
    // 执行数据库迁移
    database.Migrate()
    
    // 其他应用程序逻辑...
}
```

### 从其他包访问数据库连接

其他包可以通过 `database.GetDB()` 函数获取数据库连接实例：

```go
import (
    "go_creation/database"
    "go_creation/models"
)

func SomeFunction() {
    // 获取数据库连接
    db := database.GetDB()
    
    // 使用数据库连接
    var users []models.User
    db.Find(&users)
}
```

## 环境变量

数据库模块需要以下环境变量：

- `DB_HOST`: 数据库主机地址
- `DB_PORT`: 数据库端口
- `DB_USER`: 数据库用户名
- `DB_PASSWORD`: 数据库密码
- `DB_NAME`: 数据库名称

这些环境变量可以在 `.env` 文件中设置，或者通过系统环境变量设置。 