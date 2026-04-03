package middleware

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	EmailKey  contextKey = "email"
	ClaimsKey contextKey = "supabase_claims"
)

type SupabaseClaims struct {
	Email        string                 `json:"email"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	jwt.RegisteredClaims
}

type Verifier struct {
	jwtSecret []byte
	issuer    string
	jwks      keyfunc.Keyfunc
}

func NewVerifierFromEnv() (*Verifier, error) {
	supabaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SUPABASE_URL")), "/")
	jwtSecret := strings.TrimSpace(os.Getenv("SUPABASE_JWT_SECRET"))
	if supabaseURL == "" || jwtSecret == "" {
		return nil, errors.New("SUPABASE_URL and SUPABASE_JWT_SECRET are required")
	}

	jwtSecretBytes, err := base64.StdEncoding.DecodeString(jwtSecret)
	if err != nil {
		// Fallback to raw bytes if it's not base64 encoded
		jwtSecretBytes = []byte(jwtSecret)
	}

	jwksURL := supabaseURL + "/auth/v1/.well-known/jwks.json"
	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		fmt.Printf("warning: failed to load jwks: %v\n", err)
		k = nil
	}

	return &Verifier{
		jwtSecret: jwtSecretBytes,
		issuer:    supabaseURL + "/auth/v1",
		jwks:      k,
	}, nil
}

// ParseAndValidateAccessToken verifies Supabase-issued JWT and returns typed claims.
func (v *Verifier) ParseAndValidateAccessToken(tokenString string) (*SupabaseClaims, error) {
	claims := &SupabaseClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		switch token.Method.Alg() {
		case jwt.SigningMethodHS256.Alg():
			return v.jwtSecret, nil
		case jwt.SigningMethodRS256.Alg(), jwt.SigningMethodES256.Alg():
			if v.jwks == nil {
				return nil, fmt.Errorf("JWKS not initialized for %s token", token.Method.Alg())
			}
			return v.jwks.Keyfunc(token)
		default:
			return nil, fmt.Errorf("unexpected signing algorithm: %s", token.Method.Alg())
		}
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("token expired")
		}
		return nil, fmt.Errorf("invalid token: %v", err)
	}
	if token == nil || !token.Valid {
		return nil, errors.New("invalid token: valid is false")
	}
	if claims.Subject == "" {
		return nil, errors.New("invalid token subject")
	}
	if claims.Issuer != v.issuer {
		return nil, fmt.Errorf("invalid token issuer: expected %s got %s", v.issuer, claims.Issuer)
	}
	if !containsAudience(claims.Audience, "authenticated") {
		return nil, fmt.Errorf("invalid token audience: expected authenticated got %v", claims.Audience)
	}
	return claims, nil
}

// RequireAuth validates Authorization header, verifies token, and injects identity into typed request context keys.
func RequireAuth(verifier *Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "empty bearer token"})
			return
		}

		claims, err := verifier.ParseAndValidateAccessToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims.Subject)
		ctx = context.WithValue(ctx, EmailKey, claims.Email)
		ctx = context.WithValue(ctx, ClaimsKey, claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func containsAudience(aud jwt.ClaimStrings, expected string) bool {
	for _, value := range aud {
		if strings.TrimSpace(value) == expected {
			return true
		}
	}
	return false
}
