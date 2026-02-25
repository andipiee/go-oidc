package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type JWTService struct {
	issuer          string
	accessTokenTTL  int64
	refreshTokenTTL int64
	privateKey      *rsa.PrivateKey
	publicKey       *rsa.PublicKey
}

type Claims struct {
	jwt.RegisteredClaims
	Scope    string `json:"scope"`
	ClientID string `json:"client_id"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
	Picture  string `json:"picture,omitempty"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewJWTService(issuer string, accessTokenTTL, refreshTokenTTL int64) *JWTService {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	return &JWTService{
		issuer:          issuer,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		privateKey:      privateKey,
		publicKey:       &privateKey.PublicKey,
	}
}

func (s *JWTService) GenerateAccessToken(userID, clientID, scope, email, name, picture string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.accessTokenTTL) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			ID:        generateRandomString(32),
		},
		Scope:    scope,
		ClientID: clientID,
		Email:    email,
		Name:     name,
		Picture:  picture,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "1"
	return token.SignedString(s.privateKey)
}

func (s *JWTService) GenerateRefreshToken(userID, clientID, scope string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.refreshTokenTTL) * time.Second)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    s.issuer,
		ID:        generateRandomString(32),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *JWTService) GetJWKS() JWKS {
	return JWKS{
		Keys: []JWK{
			{
				Kty: "RSA",
				Use: "sig",
				Kid: "1",
				Alg: "RS256",
				N:   base64.RawURLEncoding.EncodeToString(s.publicKey.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(s.publicKey.E)).Bytes()),
			},
		},
	}
}

func (s *JWTService) GetJWKSJSON() ([]byte, error) {
	return json.Marshal(s.GetJWKS())
}

func (s *JWTService) HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

type CryptoService struct{}

func NewCryptoService() *CryptoService {
	return &CryptoService{}
}

func (s *CryptoService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *CryptoService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *CryptoService) GenerateRandomString(length int) string {
	return generateRandomString(length)
}

func (s *CryptoService) GenerateUserCode() string {
	return generateUserCode()
}

func (s *CryptoService) GenerateDeviceCode() string {
	return generateRandomString(64)
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:length]
}

func generateUserCode() string {
	const charset = "BCDFGHJKLMNPQRSTVWXYZ23456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[randIntn(len(charset))]
	}
	return string(b)
}

func randIntn(n int) int {
	b := make([]byte, 1)
	rand.Read(b)
	return int(b[0]) % n
}

func (s *CryptoService) VerifyCodeChallenge(codeVerifier, codeChallenge, method string) bool {
	if method == "S256" {
		hash := sha256.Sum256([]byte(codeVerifier))
		computed := base64.RawURLEncoding.EncodeToString(hash[:])
		return computed == codeChallenge
	}
	return codeVerifier == codeChallenge
}
