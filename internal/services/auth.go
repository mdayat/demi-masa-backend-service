package services

import (
	"context"
	"fmt"

	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/avast/retry-go/v4"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/repository"
	"google.golang.org/api/idtoken"
)

type AuthServicer interface {
	ValidateIDToken(ctx context.Context, idToken string) (*idtoken.Payload, error)
	CheckUserExistence(ctx context.Context, userId string) (bool, error)
	CreateRefreshToken(userId string) (CreateRefreshTokenResult, error)
	ValidateRefreshToken(tokenString string) (string, error)
	CreateAccessToken(userId string) (string, error)
	ValidateAccessToken(tokenString string) (string, error)
	SelectUserById(ctx context.Context, userId string) (repository.User, error)
	InsertRefreshToken(ctx context.Context, arg repository.InsertRefreshTokenParams) error
}

type auth struct {
	configs configs.Configs
}

func NewAuthService(configs configs.Configs) AuthServicer {
	return &auth{
		configs: configs,
	}
}

func (a auth) ValidateIDToken(ctx context.Context, idToken string) (*idtoken.Payload, error) {
	validator, err := idtoken.NewValidator(ctx)
	if err != nil {
		return nil, err
	}

	return validator.Validate(ctx, idToken, a.configs.Env.ClientId)
}

func (a auth) CheckUserExistence(ctx context.Context, userId string) (bool, error) {
	return retry.DoWithData(
		func() (bool, error) {
			return a.configs.Db.Queries.CheckUserExistence(ctx, userId)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}

type CreateUserParams struct {
	UserId       string
	RefreshToken string
}

type TokenType int

const (
	Refresh TokenType = iota
	Access
)

type RefreshTokenClaims struct {
	Type TokenType `json:"type"`
	jwt.RegisteredClaims
}

type AccessTokenClaims struct {
	Type TokenType `json:"type"`
	jwt.RegisteredClaims
}

type CreateRefreshTokenResult struct {
	Claims      RefreshTokenClaims
	TokenString string
}

func (a auth) CreateRefreshToken(userId string) (CreateRefreshTokenResult, error) {
	now := time.Now()
	claims := RefreshTokenClaims{
		Type: Refresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    a.configs.Env.OriginURL,
			Subject:   userId,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.configs.Env.SecretKey)
	if err != nil {
		return CreateRefreshTokenResult{}, nil
	}

	return CreateRefreshTokenResult{claims, tokenString}, nil
}

func (a auth) ValidateRefreshToken(tokenString string) (string, error) {
	token, err := jwt.Parse(
		tokenString,
		func(_ *jwt.Token) (interface{}, error) {
			return a.configs.Env.SecretKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithIssuer(a.configs.Env.OriginURL),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	)

	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok {
		return "", fmt.Errorf("invalid refresh token claims: %w", err)
	}

	if claims.Type != Refresh {
		return "", fmt.Errorf("invalid refresh token type: %w", err)
	}

	return claims.Subject, nil
}

func (a auth) CreateAccessToken(userId string) (string, error) {
	now := time.Now()
	claims := AccessTokenClaims{
		Type: Access,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    a.configs.Env.OriginURL,
			Subject:   userId,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.configs.Env.SecretKey)
}

func (a auth) ValidateAccessToken(tokenString string) (string, error) {
	token, err := jwt.Parse(
		tokenString,
		func(_ *jwt.Token) (interface{}, error) {
			return a.configs.Env.SecretKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithIssuer(a.configs.Env.OriginURL),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	)

	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid access token: %w", err)
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok {
		return "", fmt.Errorf("invalid access token claims: %w", err)
	}

	if claims.Type != Access {
		return "", fmt.Errorf("invalid access token type: %w", err)
	}

	return claims.Subject, nil
}

func (a auth) SelectUserById(ctx context.Context, userId string) (repository.User, error) {
	return retry.DoWithData(
		func() (repository.User, error) {
			return a.configs.Db.Queries.SelectUserById(ctx, userId)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}

func (a auth) InsertRefreshToken(ctx context.Context, arg repository.InsertRefreshTokenParams) error {
	return retry.Do(
		func() error {
			return a.configs.Db.Queries.InsertRefreshToken(ctx, arg)
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}
