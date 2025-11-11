package handlers

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ResponseData struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// TokenResponse es la estructura de datos devuelta en un login/registro exitoso.
type TokenResponse struct {
	Token     string `json:"token"`
	IDPersona int    `json:"id_persona"`
	Email     string `json:"email"`
	Nombre    string `json:"nombre"`
}

type Claims struct {
	IDPersona int    `json:"id_persona"`
	Email     string `json:"email"`
	jwt.RegisteredClaims
}

// JWT Functions

func GenerateToken(idPersona int, email string) (string, error) {
	secretKey := os.Getenv("JWT_SECRET")

	expirationTime := time.Now().Add(7 * 24 * time.Hour)

	claims := &Claims{
		IDPersona: idPersona,
		Email:     email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func VerifyToken(tokenString string) (*Claims, error) {
	secretKey := os.Getenv("JWT_SECRET")

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("token inválido")
	}

	return claims, nil
}
