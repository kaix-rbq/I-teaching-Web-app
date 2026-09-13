// Package jwtutil 负责 JWT 的签发与解析（HS256）。
// claims 保持最小化：sub=user_id、role、did=部门 id；不放部门名等可变信息。
package jwtutil

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是「爱教学」使用的 JWT 载荷。
type Claims struct {
	Role   string `json:"role"`
	DeptID uint64 `json:"did"`
	jwt.RegisteredClaims
}

// UserID 从 Subject 解析出用户 id。
func (c *Claims) UserID() (uint64, error) {
	if c.Subject == "" {
		return 0, errors.New("jwtutil: subject 为空")
	}
	id, err := strconv.ParseUint(c.Subject, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("jwtutil: subject 非法: %w", err)
	}
	return id, nil
}

// Manager 使用固定密钥签发与校验令牌。
type Manager struct {
	secret []byte
	ttl    time.Duration
}

// New 构造签发器。
func New(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

// Sign 为指定用户签发令牌。
func (m *Manager) Sign(userID uint64, role string, deptID uint64, now time.Time) (string, error) {
	claims := Claims{
		Role:   role,
		DeptID: deptID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("jwtutil: 签发失败: %w", err)
	}
	return signed, nil
}

// Parse 校验并解析令牌。
func (m *Manager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwtutil: 非法签名算法 %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("jwtutil: 解析失败: %w", err)
	}
	return claims, nil
}
