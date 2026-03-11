package auth

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenDuration = 8 * time.Hour

type Claims struct {
	GithubLogin string `json:"github_login"`
	GithubID    int64  `json:"github_id"`
	Role        string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateJWT(login string, githubID int64, role string) (string, error) {
	log.Printf("[JWT][GENERATE] Generating token → login=%s githubID=%d role=%s", login, githubID, role)

	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		log.Println("[JWT][GENERATE][ERROR] JWT_SECRET env var is empty — cannot sign token")
		return "", errors.New("JWT_SECRET is not set")
	}

	expiresAt := time.Now().Add(tokenDuration)

	claims := Claims{
		GithubLogin: login,
		GithubID:    githubID,
		Role:        role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   login,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(secret)
	if err != nil {
		log.Printf("[JWT][GENERATE][ERROR] Failed to sign token for login=%s: %v", login, err)
		return "", err
	}

	log.Printf("[JWT][GENERATE][SUCCESS] Token issued → login=%s role=%s expiresAt=%s",
		login, role, expiresAt.Format(time.RFC3339))
	return signed, nil
}

func ValidateJWT(tokenStr string) (*Claims, error) {
	log.Println("[JWT][VALIDATE] Validating incoming JWT")

	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		log.Println("[JWT][VALIDATE][ERROR] JWT_SECRET env var is empty — cannot validate token")
		return nil, errors.New("JWT_SECRET is not set")
	}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				algo := t.Header["alg"]
				log.Printf("[JWT][VALIDATE][ERROR] Unexpected signing algorithm: %v", algo)
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		},
	)

	if err != nil {
		log.Printf("[JWT][VALIDATE][ERROR] Token parse/validation failed: %v", err)
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		log.Println("[JWT][VALIDATE][ERROR] Token claims invalid or token marked invalid")
		return nil, errors.New("invalid token")
	}

	remaining := time.Until(claims.ExpiresAt.Time).Round(time.Minute)
	log.Printf("[JWT][VALIDATE][SUCCESS] Token valid → login=%s role=%s expiresIn=%s",
		claims.GithubLogin, claims.Role, remaining)

	return claims, nil
}