package configs

import (
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	DatabaseURL        string
	AllowedOrigins     string
	SecretKey          string
	OriginURL          string
	TripayMerchantCode string
	TripayAPIKey       string
	TripayPrivateKey   string
	GeoapifyAPIKey     string
}

func LoadEnv(filenames ...string) (Env, error) {
	if err := godotenv.Load(filenames...); err != nil {
		return Env{}, err
	}

	env := Env{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		AllowedOrigins:     os.Getenv("ALLOWED_ORIGINS"),
		SecretKey:          os.Getenv("SECRET_KEY"),
		OriginURL:          os.Getenv("ORIGIN_URL"),
		TripayMerchantCode: os.Getenv("TRIPAY_MERCHANT_CODE"),
		TripayAPIKey:       os.Getenv("TRIPAY_API_KEY"),
		TripayPrivateKey:   os.Getenv("TRIPAY_PRIVATE_KEY"),
		GeoapifyAPIKey:     os.Getenv("GEOAPIFY_API_KEY"),
	}

	return env, nil
}
