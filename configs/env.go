package configs

import (
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	DatabaseURL    string
	AllowedOrigins string
	ClientId       string
}

func LoadEnv() (Env, error) {
	if err := godotenv.Load(); err != nil {
		return Env{}, err
	}

	env := Env{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		AllowedOrigins: os.Getenv("ALLOWED_ORIGINS"),
		ClientId:       os.Getenv("CLIENT_ID"),
	}

	return env, nil
}
