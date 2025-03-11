package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gofiber/fiber/v2"
)

// 从环境变量获取JWT密钥，如果未设置则使用随机生成的密钥
// 在生产环境中，应确保设置了环境变量JWT_SECRET
var jwtSecret = getJWTSecret()

// getJWTSecret 从环境变量获取JWT密钥
// 如果环境变量未设置，则生成随机密钥（仅用于开发环境）
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// 检查当前环境
		env := os.Getenv("ENV")
		if env == "production" {
			log.Fatal("在生产环境中必须设置JWT_SECRET环境变量")
		}

		// 在开发环境中，生成随机密钥
		log.Println("警告: JWT_SECRET环境变量未设置，将使用随机生成的密钥（仅用于开发环境）")

		// 生成32字节的随机密钥
		randomKey := make([]byte, 32)
		if _, err := rand.Read(randomKey); err != nil {
			log.Printf("生成随机密钥失败: %v，将使用备用密钥", err)
			// 使用备用密钥，但仍然比原来的默认密钥更强
			return []byte("go_creation_jwt_secret_key_for_development_only_do_not_use_in_production_environment")
		}

		// 将随机字节编码为base64字符串
		secret = base64.StdEncoding.EncodeToString(randomKey)
		log.Printf("已生成随机JWT密钥: %s", secret)
	}

	// 确保密钥长度足够
	if len(secret) < 16 {
		log.Println("警告: JWT密钥长度不足，建议使用至少32字符的密钥")
	}

	return []byte(secret)
}

// SalespersonClaims 定义JWT令牌的声明结构
// 包含销售人员的身份信息和标准JWT声明
type SalespersonClaims struct {
	SalespersonID        uint   `json:"salesperson_id"` // 销售人员ID，用于身份识别
	Username             string `json:"username"`       // 销售人员用户名，用于日志和审计
	jwt.RegisteredClaims        // 嵌入标准JWT声明（如过期时间、签发时间等）
}

// GenerateToken 生成JWT令牌
// 该函数为指定的销售人员创建一个签名的JWT令牌
// 参数:
//   - salespersonID: 销售人员的唯一标识符
//   - username: 销售人员的用户名
//   - duration: 令牌的有效期限
//
// 返回:
//   - string: 生成的JWT令牌字符串
//   - error: 如果令牌生成过程中发生错误
func GenerateToken(salespersonID uint, username string, duration time.Duration) (string, error) {
	// 设置令牌过期时间
	expirationTime := time.Now().Add(duration)

	// 创建JWT声明
	claims := SalespersonClaims{
		SalespersonID: salespersonID,
		Username:      username,
		RegisteredClaims: jwt.RegisteredClaims{
			// 令牌过期时间
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			// 令牌签发时间
			IssuedAt: jwt.NewNumericDate(time.Now()),
			// 令牌生效时间
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// 创建令牌对象并使用HS256算法签名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用密钥签名令牌并获取完整的签名字符串
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseToken 解析并验证JWT令牌
// 该函数验证令牌的签名并提取其中的声明信息
// 参数:
//   - tokenString: 要解析的JWT令牌字符串
//
// 返回:
//   - *SalespersonClaims: 令牌中包含的销售人员声明信息
//   - error: 如果令牌无效或解析过程中发生错误
func ParseToken(tokenString string) (*SalespersonClaims, error) {
	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &SalespersonClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return jwtSecret, nil
	})

	// 处理解析错误
	if err != nil {
		return nil, err
	}

	// 验证令牌有效性并提取声明
	if claims, ok := token.Claims.(*SalespersonClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的令牌")
}

// GetSalespersonIDFromToken 从Fiber上下文中获取销售员ID
// 该函数从请求的Authorization头中提取JWT令牌，解析并返回销售员ID
// 参数:
//   - c: Fiber上下文，包含请求信息
//
// 返回:
//   - uint: 销售员ID
//   - error: 如果令牌无效或解析过程中发生错误
func GetSalespersonIDFromToken(c *fiber.Ctx) (uint, error) {
	// 从上下文中获取销售员ID
	salespersonID := c.Locals("salesperson_id")
	
	// 如果已经在上下文中存在，直接返回
	if id, ok := salespersonID.(uint); ok {
		return id, nil
	}
	
	// 从请求头中获取令牌
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return 0, errors.New("未提供授权令牌")
	}
	
	// 检查令牌格式
	if len(authHeader) <= 7 || authHeader[:7] != "Bearer " {
		return 0, errors.New("无效的授权令牌格式")
	}
	
	// 提取令牌字符串
	tokenString := authHeader[7:]
	
	// 解析令牌
	claims, err := ParseToken(tokenString)
	if err != nil {
		return 0, err
	}
	
	// 返回销售员ID
	return claims.SalespersonID, nil
}
