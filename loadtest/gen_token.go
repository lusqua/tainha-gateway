//go:build ignore

package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": "loadtest",
		"role":     "admin",
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})
	s, _ := token.SignedString([]byte("e2e-test-secret"))
	fmt.Println(s)
}
