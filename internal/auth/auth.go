package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrNoAuthHeaderIncluded = errors.New("no authentication header included in request")
var ErrNoAuthKeyIncluded = errors.New("no authentication key included in request")

func HashPassword(pw string) (string, error) {
	return argon2id.CreateHash(pw, argon2id.DefaultParams)
}

func CheckPasswordHash(pw, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(pw, hash)
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	signingKey := []byte(tokenSecret)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "chirpy-access",
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject: userID.String(),
	})

	return token.SignedString(signingKey)
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := jwt.RegisteredClaims{}
	keyFunc := func(token *jwt.Token) (any, error) {
		 return []byte(tokenSecret), nil
	}
	token, err := jwt.ParseWithClaims(tokenString, &claims, keyFunc)
	if err != nil {
		return uuid.Nil, err 
	}

	rawUUID, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}
	
	userId, err :=  uuid.Parse(rawUUID)
	if err != nil {
		return uuid.Nil, err
	}

	return userId, nil
}

func GetBearerToken(h http.Header) (string, error) {
	authHeader := h.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoAuthHeaderIncluded
	}
	splitAuth := strings.Split(authHeader, " ")
	if len(splitAuth) < 2 || splitAuth[0] != "Bearer" {
		return "", errors.New("Malformeed authorization header")
	}

	return splitAuth[1], nil
}

func MakeRefreshToken() string {
	key := make([]byte, 32)
	rand.Read(key)
	fmt.Printf("% x\n", key)
	return hex.EncodeToString(key)
}

func GetAPIKey(headers http.Header) (string, error) {
	key := headers.Get("Authorization")
	if key == "" {
		return "", ErrNoAuthKeyIncluded
	}
	splitKey := strings.Split(key, " ")
	if len(splitKey) < 2 || splitKey[0] != "ApiKey" {
		return "", errors.New("Malformeed authorization key")
	}

	return splitKey[1], nil
}