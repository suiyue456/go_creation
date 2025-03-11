package utils

import (
	"crypto/rand"
	mathrand "math/rand"
	"strconv"
	"sync/atomic"
	"time"
)

// 字符集常量
const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// 全局原子计数器，用于确保生成的代码唯一
var codeCounter int64

// GenerateRandomCode 生成指定长度的随机字符码
func GenerateRandomCode(length int) string {
	code := make([]byte, length)

	// 使用安全的随机数生成
	_, err := rand.Read(code)
	if err != nil {
		// 如果安全随机数生成失败，回退到不安全的方法
		// 创建一个新的随机数生成器实例，而不是使用全局的Seed
		r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
		for i := range code {
			code[i] = charset[r.Intn(len(charset))]
		}
		return string(code)
	}

	// 将随机字节映射到字符集
	for i := range code {
		code[i] = charset[int(code[i])%len(charset)]
	}

	return string(code)
}

// GenerateInviteCode 生成邀请码
func GenerateInviteCode() string {
	return GenerateRandomCode(8)
}

// GenerateAgentCode 生成代理码
func GenerateAgentCode() string {
	return GenerateRandomCode(6)
}

// GenerateSalespersonKeyCode 生成销售员密钥码
func GenerateSalespersonKeyCode() string {
	// 使用原子计数器确保唯一性
	counter := atomic.AddInt64(&codeCounter, 1)
	// 添加4位随机字符以增加唯一性
	randomPart := GenerateRandomCode(4)
	return "KEY" + strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.FormatInt(counter, 36) + randomPart
}

// GenerateSalespersonCode 生成销售员卡密码
func GenerateSalespersonCode() string {
	// 使用原子计数器确保唯一性
	counter := atomic.AddInt64(&codeCounter, 1)
	// 添加4位随机字符以增加唯一性
	randomPart := GenerateRandomCode(4)
	return "CODE" + strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.FormatInt(counter, 36) + randomPart
}
