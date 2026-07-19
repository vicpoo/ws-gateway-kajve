//src/infrastructure/auth/jwt.go
// Package auth valida los JWT que ya emite api-mobile. Este servicio nunca
// firma tokens, solo los valida — deben usar exactamente el mismo secreto
// HS256 que api-mobile (config.JWTSecret allá, en
// internal/application/usecases/auth_service.go).
package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// Claims son los campos que nos interesan del JWT de api-mobile (allá viven
// en internal/domain/entities.JWTClaims). Los tokens se generan ahí con
// jwt.MapClaims{"user_id", "email", "rol", "exp", "iat", "type"} — el campo
// Type aquí nos deja rechazar refresh tokens al conectar.
type Claims struct {
	jwt.RegisteredClaims
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Rol    string `json:"rol"`
	Type   string `json:"type"`
}

// Validator valida JWTs HS256 firmados con un secreto compartido.
type Validator struct {
	secret []byte
}

func NewValidator(secret string) *Validator {
	return &Validator{secret: []byte(secret)}
}

// Validate valida firma y expiración, y rechaza tokens que no sean de tipo
// "access" (un refresh token no debe poder abrir un WebSocket).
func (v *Validator) Validate(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: método de firma inesperado: %v", t.Header["alg"])
		}
		return v.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: error parseando token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("auth: token inválido")
	}
	if claims.Type == "refresh" {
		return nil, errors.New("auth: no se puede usar un refresh token para conectar")
	}
	return claims, nil
}
