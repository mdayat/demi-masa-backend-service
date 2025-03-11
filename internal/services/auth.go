package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/argon2"

	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/dbutil"
	"github.com/mdayat/demi-masa-backend-service/internal/retryutil"
	"github.com/mdayat/demi-masa-backend-service/repository"
	"google.golang.org/api/idtoken"
)

type AuthServicer interface {
	ValidateIDToken(ctx context.Context, idToken string) (validateIDTokenResult, error)
	CreateRefreshToken(claims RefreshTokenClaims) (string, error)
	ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error)
	CreateAccessToken(claims AccessTokenClaims) (string, error)
	ValidateAccessToken(tokenString string) (*AccessTokenClaims, error)
	RotateRefreshToken(ctx context.Context, arg RotateRefreshTokenParams) (rotateRefreshTokenResult, error)
	RegisterUser(ctx context.Context, arg RegisterUserParams) (registerUserResult, error)
	AuthenticateUser(ctx context.Context, arg AuthenticateUserParams) (authenticateUserResult, error)
}

type auth struct {
	configs configs.Configs
}

func NewAuthService(configs configs.Configs) AuthServicer {
	return &auth{
		configs: configs,
	}
}

type validateIDTokenResult struct {
	Subject string
	Email   string
	Name    string
}

func (a auth) ValidateIDToken(ctx context.Context, idToken string) (validateIDTokenResult, error) {
	validator, err := idtoken.NewValidator(ctx)
	if err != nil {
		return validateIDTokenResult{}, err
	}

	payload, err := validator.Validate(ctx, idToken, a.configs.Env.ClientId)
	if err != nil {
		return validateIDTokenResult{}, fmt.Errorf("failed to validate Id token: %w", err)
	}

	emailRaw, exists := payload.Claims["email"]
	if !exists {
		return validateIDTokenResult{}, errors.New("email claim is missing")
	}

	email, ok := emailRaw.(string)
	if !ok {
		return validateIDTokenResult{}, errors.New("email claim is not a string")
	}

	nameRaw, exists := payload.Claims["name"]
	if !exists {
		return validateIDTokenResult{}, errors.New("name claim is missing")
	}

	name, ok := nameRaw.(string)
	if !ok {
		return validateIDTokenResult{}, errors.New("name claim is not a string")
	}

	validateIDTokenResult := validateIDTokenResult{
		Subject: payload.Subject,
		Email:   email,
		Name:    name,
	}

	return validateIDTokenResult, nil
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
	return token.SignedString([]byte(a.configs.Env.SecretKey))
}

func (a auth) ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&RefreshTokenClaims{},
		func(_ *jwt.Token) (interface{}, error) {
			return []byte(a.configs.Env.SecretKey), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithIssuer(a.configs.Env.OriginURL),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok {
		return nil, errors.New("invalid refresh token claims")
	}

	if claims.Type != Refresh {
		return nil, errors.New("invalid refresh token type")
	}

	return claims, nil
}

type AccessTokenClaims struct {
	Type TokenType `json:"type"`
	jwt.RegisteredClaims
}

func (a auth) CreateAccessToken(claims AccessTokenClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.configs.Env.SecretKey))
}

func (a auth) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&AccessTokenClaims{},
		func(_ *jwt.Token) (interface{}, error) {
			return []byte(a.configs.Env.SecretKey), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithIssuer(a.configs.Env.OriginURL),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid access token")
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok {
		return nil, errors.New("invalid access token claims")
	}

	if claims.Type != Access {
		return nil, errors.New("invalid access token type")
	}

	return claims, nil
}

type RotateRefreshTokenParams struct {
	Jti       string
	UserUUID  pgtype.UUID
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
			UserID: arg.UserUUID,
		})

		if err != nil {
			return rotateRefreshTokenResult{}, fmt.Errorf("failed to revoke refresh token: %w", err)
		}

		userId := arg.UserUUID.String()
		now := time.Now()
		remainingExpiration := arg.ExpiresAt.Sub(now)

		refreshTokenClaims := RefreshTokenClaims{
			Type: Refresh,
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        uuid.NewString(),
				ExpiresAt: jwt.NewNumericDate(now.Add(remainingExpiration)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    a.configs.Env.OriginURL,
				Subject:   userId,
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
				Subject:   userId,
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
			UserID:    arg.UserUUID,
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

type prayerName string

const (
	subuh  prayerName = "subuh"
	zuhur  prayerName = "zuhur"
	asar   prayerName = "asar"
	magrib prayerName = "magrib"
	isya   prayerName = "isya"
)

func (a auth) createInsertPrayersParams(userUUID pgtype.UUID) []repository.InsertPrayersParams {
	now := time.Now()
	firstDayOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	nextMonth := firstDayOfThisMonth.AddDate(0, 1, 0)
	lastDayOfThisMonth := nextMonth.AddDate(0, 0, -1)
	numOfDaysOfThisMonth := lastDayOfThisMonth.Day()

	numOfPrayersDaily := 5
	insertPrayersParams := make([]repository.InsertPrayersParams, 0, numOfDaysOfThisMonth*numOfPrayersDaily)

	var prayerName prayerName
	for day := now.Day(); day <= numOfDaysOfThisMonth; day++ {
		for i := 1; i <= numOfPrayersDaily; i++ {
			switch i {
			case 1:
				prayerName = subuh
			case 2:
				prayerName = zuhur
			case 3:
				prayerName = asar
			case 4:
				prayerName = magrib
			case 5:
				prayerName = isya
			}

			insertPrayersParams = append(insertPrayersParams, repository.InsertPrayersParams{
				ID:     pgtype.UUID{Bytes: uuid.New(), Valid: true},
				UserID: userUUID,
				Name:   string(prayerName),
				Year:   int16(now.Year()),
				Month:  int16(now.Month()),
				Day:    int16(day),
			})
		}
	}

	return insertPrayersParams
}

func (a auth) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	var iterations uint32 = 3
	var memory uint32 = 64 * 1024
	var parallelism uint8 = 4
	var keyLength uint32 = 32

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	finalHash := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memory,
		iterations,
		parallelism,
		encodedSalt,
		encodedHash,
	)

	return finalHash, nil
}

type RegisterUserParams struct {
	UserUUID  pgtype.UUID
	Username  string
	UserEmail string
	Password  string
}

type registerUserResult struct {
	User         repository.User
	RefreshToken string
	AccessToken  string
}

func (a auth) RegisterUser(ctx context.Context, arg RegisterUserParams) (registerUserResult, error) {
	hashedPassword, err := a.hashPassword(arg.Password)
	if err != nil {
		return registerUserResult{}, fmt.Errorf("failed to hash password: %w", err)
	}

	retryableFunc := func(qtx *repository.Queries) (registerUserResult, error) {
		user, err := qtx.InsertUser(ctx, repository.InsertUserParams{
			ID:          arg.UserUUID,
			Name:        arg.Username,
			Email:       arg.UserEmail,
			Password:    hashedPassword,
			Coordinates: pgtype.Point{P: pgtype.Vec2{X: 106.865036, Y: -6.175110}, Valid: true},
			City:        "Jakarta",
			Timezone:    "Asia/Jakarta",
		})

		if err != nil {
			return registerUserResult{}, fmt.Errorf("failed to insert user: %w", err)
		}

		insertPrayersParams := a.createInsertPrayersParams(user.ID)
		_, err = qtx.InsertPrayers(ctx, insertPrayersParams)
		if err != nil {
			return registerUserResult{}, fmt.Errorf("failed to insert prayers: %w", err)
		}

		userId := user.ID.String()
		now := time.Now()

		refreshTokenClaims := RefreshTokenClaims{
			Type: Refresh,
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        uuid.NewString(),
				ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    a.configs.Env.OriginURL,
				Subject:   userId,
			},
		}

		refreshToken, err := a.CreateRefreshToken(refreshTokenClaims)
		if err != nil {
			return registerUserResult{}, fmt.Errorf("failed to create refresh token: %w", err)
		}

		accessTokenClaims := AccessTokenClaims{
			Type: Access,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    a.configs.Env.OriginURL,
				Subject:   userId,
			},
		}

		accessToken, err := a.CreateAccessToken(accessTokenClaims)
		if err != nil {
			return registerUserResult{}, fmt.Errorf("failed to create access token: %w", err)
		}

		refreshTokenUUID, err := uuid.Parse(refreshTokenClaims.ID)
		if err != nil {
			return registerUserResult{}, fmt.Errorf("failed to parse JTI to UUID: %w", err)
		}

		err = qtx.InsertRefreshToken(ctx, repository.InsertRefreshTokenParams{
			ID:        pgtype.UUID{Bytes: refreshTokenUUID, Valid: true},
			UserID:    user.ID,
			ExpiresAt: pgtype.Timestamptz{Time: refreshTokenClaims.ExpiresAt.Time, Valid: true},
		})

		if err != nil {
			return registerUserResult{}, fmt.Errorf("failed to insert refresh token: %w", err)
		}

		registerUserResult := registerUserResult{
			User:         user,
			RefreshToken: refreshToken,
			AccessToken:  accessToken,
		}

		return registerUserResult, nil
	}

	return dbutil.RetryableTxWithData(ctx, a.configs.Db.Conn, a.configs.Db.Queries, retryableFunc)
}

type AuthenticateUserParams struct {
	Email    string
	Password string
}

type authenticateUserResult struct {
	User         repository.User
	RefreshToken string
	AccessToken  string
}

func (a auth) AuthenticateUser(ctx context.Context, arg AuthenticateUserParams) (authenticateUserResult, error) {
	hashedPassword, err := a.hashPassword(arg.Password)
	if err != nil {
		return authenticateUserResult{}, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := retryutil.RetryWithData(func() (repository.User, error) {
		return a.configs.Db.Queries.SelectUserByEmailAndPassword(ctx, repository.SelectUserByEmailAndPasswordParams{
			Email:    arg.Email,
			Password: hashedPassword,
		})
	})

	if err != nil {
		return authenticateUserResult{}, fmt.Errorf("failed to select user by email and password: %w", err)
	}

	userId := user.ID.String()
	now := time.Now()

	refreshTokenClaims := RefreshTokenClaims{
		Type: Refresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    a.configs.Env.OriginURL,
			Subject:   userId,
		},
	}

	refreshToken, err := a.CreateRefreshToken(refreshTokenClaims)
	if err != nil {
		return authenticateUserResult{}, fmt.Errorf("failed to create refresh token: %w", err)
	}

	accessTokenClaims := AccessTokenClaims{
		Type: Access,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    a.configs.Env.OriginURL,
			Subject:   userId,
		},
	}

	accessToken, err := a.CreateAccessToken(accessTokenClaims)
	if err != nil {
		return authenticateUserResult{}, fmt.Errorf("failed to create access token: %w", err)
	}

	refreshTokenUUID, err := uuid.Parse(refreshTokenClaims.ID)
	if err != nil {
		return authenticateUserResult{}, fmt.Errorf("failed to parse JTI to UUID: %w", err)
	}

	err = retryutil.RetryWithoutData(func() error {
		return a.configs.Db.Queries.InsertRefreshToken(ctx, repository.InsertRefreshTokenParams{
			ID:        pgtype.UUID{Bytes: refreshTokenUUID, Valid: true},
			UserID:    user.ID,
			ExpiresAt: pgtype.Timestamptz{Time: refreshTokenClaims.ExpiresAt.Time, Valid: true},
		})
	})

	if err != nil {
		return authenticateUserResult{}, fmt.Errorf("failed to insert refresh token: %w", err)
	}

	authenticateUserResult := authenticateUserResult{
		User:         user,
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}

	return authenticateUserResult, nil
}
