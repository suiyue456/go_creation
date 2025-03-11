package utils

import (
	"sync"
	"time"
)

// 登录尝试信息
type LoginAttemptInfo struct {
	Count     int       // 尝试次数
	LastTry   time.Time // 最后一次尝试时间
	LockUntil time.Time // 锁定截止时间
}

// LoginLimiter 登录限制器
// 用于限制登录失败次数，防止暴力破解
type LoginLimiter struct {
	attempts      map[string]*LoginAttemptInfo // 登录尝试记录
	mutex         sync.RWMutex                 // 读写锁，保证并发安全
	maxAttempts   int                          // 最大允许的登录失败次数
	lockDuration  time.Duration                // 锁定时间
	cleanInterval time.Duration                // 清理间隔
}

// NewLoginLimiter 创建新的登录限制器
// 参数:
//   - maxAttempts: 最大允许的登录失败次数
//   - lockDuration: 锁定时间
//   - cleanInterval: 清理间隔，定期清理过期的尝试记录
func NewLoginLimiter(maxAttempts int, lockDuration, cleanInterval time.Duration) *LoginLimiter {
	limiter := &LoginLimiter{
		attempts:      make(map[string]*LoginAttemptInfo),
		maxAttempts:   maxAttempts,
		lockDuration:  lockDuration,
		cleanInterval: cleanInterval,
	}

	// 启动定期清理过期记录的协程
	go limiter.cleanupRoutine()

	return limiter
}

// cleanupRoutine 定期清理过期的尝试记录
func (l *LoginLimiter) cleanupRoutine() {
	ticker := time.NewTicker(l.cleanInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanup()
	}
}

// cleanup 清理过期的尝试记录
func (l *LoginLimiter) cleanup() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()
	for username, attempt := range l.attempts {
		// 如果锁定已过期且最后一次尝试时间超过24小时，删除记录
		if now.After(attempt.LockUntil) && now.Sub(attempt.LastTry) > 24*time.Hour {
			delete(l.attempts, username)
		}
	}
}

// RecordFailedLogin 记录登录失败
// 更新登录尝试次数，并在达到最大尝试次数时锁定账号
// 返回是否被锁定及锁定剩余时间（分钟）
func (l *LoginLimiter) RecordFailedLogin(username string) (bool, int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()

	attempt, exists := l.attempts[username]
	if !exists {
		attempt = &LoginAttemptInfo{
			Count:   0,
			LastTry: now,
		}
		l.attempts[username] = attempt
	}

	attempt.Count++
	attempt.LastTry = now

	// 如果达到最大尝试次数，锁定账号
	if attempt.Count >= l.maxAttempts {
		attempt.LockUntil = now.Add(l.lockDuration)
		return true, int(l.lockDuration.Minutes())
	}

	return false, 0
}

// IsLocked 检查账号是否被锁定
// 返回是否被锁定及锁定剩余时间（分钟）
func (l *LoginLimiter) IsLocked(username string) (bool, int) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	now := time.Now()

	attempt, exists := l.attempts[username]
	if !exists {
		return false, 0
	}

	// 如果锁定时间未过，返回锁定状态和剩余时间
	if now.Before(attempt.LockUntil) {
		remainingMinutes := int(attempt.LockUntil.Sub(now).Minutes()) + 1
		return true, remainingMinutes
	}

	return false, 0
}

// ResetAttempts 重置登录尝试次数
func (l *LoginLimiter) ResetAttempts(username string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	delete(l.attempts, username)
}

// GetRemainingAttempts 获取剩余尝试次数
func (l *LoginLimiter) GetRemainingAttempts(username string) int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	attempt, exists := l.attempts[username]
	if !exists {
		return l.maxAttempts
	}

	remaining := l.maxAttempts - attempt.Count
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// DefaultLoginLimiter 默认的登录限制器实例
// 最大尝试次数为5次，锁定时间为15分钟，每小时清理一次过期记录
var DefaultLoginLimiter = NewLoginLimiter(5, 15*time.Minute, 1*time.Hour)
