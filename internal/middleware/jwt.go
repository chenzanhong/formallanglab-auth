package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/chenzanhong/goutil/jwtx"
)

// middleware/claims.go 或直接在 middleware 包中定义
type AccessTokenClaims struct {
	Username string `json:"username" inject:"username"` // inject 到 gin.Context 的 key
	UserID   int64  `json:"user_id"       inject:"user_id"`
	jwtx.RegisteredClaims
}

func GenerateAccessToken(username string, userID int64, expireTimeSeconds int) (string, error) {
	claims := &AccessTokenClaims{
		Username: username,
		UserID:   userID,
		RegisteredClaims: jwtx.RegisteredClaims{
			ExpiresAt: jwtx.NewNumericDate(time.Now().Add(time.Duration(expireTimeSeconds) * time.Second)),
			IssuedAt:  jwtx.NewNumericDate(time.Now()),
		},
	}
	return jwtx.SignToken(claims) // 使用全局配置签名
}

// GenerateRandomRefreshToken 生成随机的刷新令牌
func GenerateRandomRefreshToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil // e.g., "a1b2c3...f9"
}
