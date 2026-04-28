package auth

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserStore struct {
	users map[string]string
}

func LoadUsers(csvPath string) (*UserStore, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("open users csv: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read users csv: %w", err)
	}

	store := &UserStore{users: make(map[string]string)}
	for i, row := range records {
		if i == 0 && (strings.EqualFold(row[0], "username") || strings.EqualFold(row[0], "user")) {
			continue
		}
		if len(row) >= 2 {
			store.users[strings.TrimSpace(row[0])] = strings.TrimSpace(row[1])
		}
	}

	return store, nil
}

func (s *UserStore) Authenticate(username, password string) bool {
	pwd, ok := s.users[username]
	return ok && pwd == password
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateToken(username, secret string) (string, error) {
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}
