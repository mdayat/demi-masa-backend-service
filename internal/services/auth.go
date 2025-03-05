package services

import (
	"context"
	"fmt"

	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/avast/retry-go/v4"
	"github.com/mdayat/demi-masa/configs"
	"github.com/mdayat/demi-masa/internal/dbutil"
	"github.com/mdayat/demi-masa/repository"
	"google.golang.org/api/idtoken"
)

type AuthServicer interface {
	ValidateIDToken(ctx context.Context, idToken string) (*idtoken.Payload, error)
	CreateRefreshToken(claims RefreshTokenClaims) (string, error)
	ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error)
	CreateAccessToken(claims AccessTokenClaims) (string, error)
	ValidateAccessToken(tokenString string) (*AccessTokenClaims, error)
	SelectUserById(ctx context.Context, userId string) (repository.User, error)
	InsertRefreshToken(ctx context.Context, arg repository.InsertRefreshTokenParams) error
	SelectRefreshTokenById(ctx context.Context, arg SelectRefreshTokenByIdParams) (repository.RefreshToken, error)
	RotateRefreshToken(ctx context.Context, arg RotateRefreshTokenParams) (rotateRefreshTokenResult, error)
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

type TokenType int

const (
	Refresh TokenType = iota
	Access
)

type RefreshTokenClaims struct {
	Type TokenType `json:"type"`
	jwt.RegisteredClaims
}

func (a auth) CreateRefreshToken(claims RefreshTokenClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.configs.Env.SecretKey)
}

func (a auth) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
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
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid refresh token claims: %w", err)
	}

	if claims.Type != Refresh {
		return nil, fmt.Errorf("invalid refresh token type: %w", err)
	}

	return claims, nil
}

type AccessTokenClaims struct {
	Type TokenType `json:"type"`
	jwt.RegisteredClaims
}

func (a auth) CreateAccessToken(claims AccessTokenClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.configs.Env.SecretKey)
}

func (a auth) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
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
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid access token claims: %w", err)
	}

	if claims.Type != Access {
		return nil, fmt.Errorf("invalid access token type: %w", err)
	}

	return claims, nil
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

type SelectRefreshTokenByIdParams struct {
	Jti    string
	UserId string
}

func (a auth) SelectRefreshTokenById(ctx context.Context, arg SelectRefreshTokenByIdParams) (repository.RefreshToken, error) {
	refreshTokenId, err := uuid.Parse(arg.Jti)
	if err != nil {
		return repository.RefreshToken{}, fmt.Errorf("failed to parse JTI to UUID: %w", err)
	}

	return retry.DoWithData(
		func() (repository.RefreshToken, error) {
			return a.configs.Db.Queries.SelectRefreshTokenById(ctx, repository.SelectRefreshTokenByIdParams{
				ID:     pgtype.UUID{Bytes: refreshTokenId, Valid: true},
				UserID: arg.UserId,
			})
		},
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)
}

type RotateRefreshTokenParams struct {
	Jti       string
	UserId    string
	ExpiresAt time.Time
}

type rotateRefreshTokenResult struct {
	RefreshToken string
	AccessToken  string
}

func (a auth) RotateRefreshToken(ctx context.Context, arg RotateRefreshTokenParams) (rotateRefreshTokenResult, error) {
	retryableFunc := func(qtx *repository.Queries) (rotateRefreshTokenResult, error) {
		oldRefreshTokenId, err := uuid.Parse(arg.Jti)
		if err != nil {
			return rotateRefreshTokenResult{}, fmt.Errorf("failed to parse old JTI to UUID: %w", err)
		}

		err = qtx.RevokeRefreshToken(ctx, repository.RevokeRefreshTokenParams{
			ID:     pgtype.UUID{Bytes: oldRefreshTokenId, Valid: true},
			UserID: arg.UserId,
		})

		if err != nil {
			return rotateRefreshTokenResult{}, fmt.Errorf("failed to revoke refresh token: %w", err)
		}

		now := time.Now()
		remainingExpiration := arg.ExpiresAt.Sub(now)

		refreshTokenClaims := RefreshTokenClaims{
			Type: Refresh,
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        uuid.NewString(),
				ExpiresAt: jwt.NewNumericDate(now.Add(remainingExpiration)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    a.configs.Env.OriginURL,
				Subject:   arg.UserId,
			},
		}

		refreshToken, err := a.CreateRefreshToken(refreshTokenClaims)
		if err != nil {
			return rotateRefreshTokenResult{}, fmt.Errorf("failed to create refresh token: %w", err)
		}

		accessTokenClaims := AccessTokenClaims{
			Type: Access,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    a.configs.Env.OriginURL,
				Subject:   arg.UserId,
			},
		}

		accessToken, err := a.CreateAccessToken(accessTokenClaims)
		if err != nil {
			return rotateRefreshTokenResult{}, fmt.Errorf("failed to create access token: %w", err)
		}

		newRefreshTokenId, err := uuid.Parse(refreshTokenClaims.ID)
		if err != nil {
			return rotateRefreshTokenResult{}, fmt.Errorf("failed to parse new JTI to UUID: %w", err)
		}

		err = qtx.InsertRefreshToken(ctx, repository.InsertRefreshTokenParams{
			ID:        pgtype.UUID{Bytes: newRefreshTokenId, Valid: true},
			UserID:    arg.UserId,
			ExpiresAt: pgtype.Timestamptz{Time: refreshTokenClaims.ExpiresAt.Time, Valid: true},
		})

		if err != nil {
			return rotateRefreshTokenResult{}, fmt.Errorf("failed to insert refresh token: %w", err)
		}

		rotateRefreshTokenResult := rotateRefreshTokenResult{
			RefreshToken: refreshToken,
			AccessToken:  accessToken,
		}

		return rotateRefreshTokenResult, nil
	}

	return dbutil.RetryableTxWithData(ctx, a.configs.Db.Conn, a.configs.Db.Queries, retryableFunc)
}
