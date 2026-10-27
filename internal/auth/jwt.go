package auth

import (
	"errors"
	"time"

	"github.com/anchel/wechat-official-account-admin/config"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID      uint   `json:"user_id"`
	TenantID    uint   `json:"tenant_id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	SSOProvider string `json:"sso_provider,omitempty"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, tenantID uint, username, role, ssoProvider string) (string, error) {
	cfg := config.GlobalConfig.JWT
	expirationTime := time.Now().Add(time.Duration(cfg.ExpireHours) * time.Hour)

	claims := &Claims{
		UserID:      userID,
		TenantID:    tenantID,
		Username:    username,
		Role:        role,
		SSOProvider: ssoProvider,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

func ParseToken(tokenString string) (*Claims, error) {
	cfg := config.GlobalConfig.JWT
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
